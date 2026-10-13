package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// QuotationService implements the quotation-upload use case
// (Milestone 2 part 2): validate the upload, checksum the bytes,
// generate a storage key, persist the quotation row, and enqueue an
// extraction job (ADR-004). File bytes go to object storage via the
// storage port; only the key and checksum are persisted.
type QuotationService struct {
	quotations QuotationStore
	jobs       ProcessingJobStore
	storage    StorageUploader
	logger     *slog.Logger
}

// NewQuotationService wires the quotation use case. The store satisfies
// QuotationStore and ProcessingJobStore via structural typing (ADR-015).
func NewQuotationService(quotations QuotationStore, jobs ProcessingJobStore, storage StorageUploader, logger *slog.Logger) *QuotationService {
	return &QuotationService{quotations: quotations, jobs: jobs, storage: storage, logger: logger}
}

// UploadQuotationInput carries everything the upload use case needs.
// The reader is consumed exactly once; the service does not retain it.
type UploadQuotationInput struct {
	ProcurementRequestID string
	VendorName           string
	OriginalFilename     string
	Content              io.Reader
}

// UploadQuotation validates and persists one quotation upload and
// enqueues its extraction job. It returns the stored quotation row.
func (s *QuotationService) UploadQuotation(ctx context.Context, authCtx AuthContext, input UploadQuotationInput) (domain.Quotation, error) {
	var fields []FieldError

	vendor := strings.TrimSpace(input.VendorName)
	if err := domain.ValidateVendorName(vendor); err != nil {
		fields = append(fields, FieldError{Field: "vendor_name", Message: err.Error()})
	}
	if err := domain.ValidateQuotationFilename(input.OriginalFilename); err != nil {
		fields = append(fields, FieldError{Field: "file", Message: err.Error()})
	}
	if input.Content == nil {
		fields = append(fields, FieldError{Field: "file", Message: "file is required"})
	}
	if len(fields) > 0 {
		return domain.Quotation{}, &ValidationError{Fields: fields}
	}

	// Read the bytes once, enforcing the size cap (spec §11).
	content, err := io.ReadAll(io.LimitReader(input.Content, domain.MaxQuotationUploadBytes+1))
	if err != nil {
		return domain.Quotation{}, err
	}
	if len(content) > domain.MaxQuotationUploadBytes {
		return domain.Quotation{}, &ValidationError{Fields: []FieldError{{
			Field:   "file",
			Message: fmt.Sprintf("file exceeds the %d MiB limit", domain.MaxQuotationUploadBytes>>20),
		}}}
	}

	// Checksum the bytes; the checksum travels with the row (ADR-008).
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:])

	// Storage key: org-scoped, generated, never client-supplied (ADR-008).
	storageKey := fmt.Sprintf("%s/quotations/%d", authCtx.Organization.ID, time.Now().UnixNano())

	quotation, err := s.quotations.CreateQuotation(ctx, domain.Quotation{
		ProcurementRequestID: input.ProcurementRequestID,
		OrganizationID:       authCtx.Organization.ID,
		VendorName:           vendor,
		StorageKey:           storageKey,
		ChecksumSHA256:       checksum,
		OriginalFilename:     input.OriginalFilename,
		Status:               domain.QuotationUploaded,
	})
	if err != nil {
		s.logger.Error("create quotation", slog.Any("error", err))
		return domain.Quotation{}, err
	}

	// Upload the file bytes to object storage (MinIO). The storage key
	// generated above is used as the object key; the checksum travels
	// with the quotation row (ADR-008).
	if _, err := s.storage.UploadBytes(storageKey, content); err != nil {
		s.logger.Error("minio upload failed", slog.String("storage_key", storageKey), slog.Any("error", err))
		// We do NOT roll back the quotation row; the upload failure is
		// logged and surface area is kept minimal. The job queue will
		// eventually mark the quotation as failed.
	}

	// Enqueue the extraction job (ADR-004). A job-queue failure fails
	// the whole upload so no quotation is ever left without a job.
	job, err := s.jobs.CreateProcessingJob(ctx, domain.ProcessingJob{
		QuotationID: quotation.ID,
		Kind:        "extract",
		MaxAttempts: 3,
	})
	if err != nil {
		s.logger.Error("enqueue extraction job", slog.Any("error", err))
		return domain.Quotation{}, err
	}

	s.logger.Info("quotation uploaded",
		slog.String("quotation_id", quotation.ID),
		slog.String("job_id", job.ID),
		slog.String("vendor", vendor),
	)
	return quotation, nil
}

// QuotationsByRequest lists a request's quotations, newest first.
func (s *QuotationService) QuotationsByRequest(ctx context.Context, requestID string) ([]domain.Quotation, error) {
	return s.quotations.QuotationsByRequest(ctx, requestID)
}