package domain

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Quotation is a vendor quotation document uploaded under a
// procurement request (spec §4). File bytes live in object storage;
// only the storage key and checksum live here.
type Quotation struct {
	ID                   string
	ProcurementRequestID string
	OrganizationID       string
	VendorName           string
	StorageKey           string
	ChecksumSHA256       string
	OriginalFilename     string
	Status               QuotationStatus
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// QuotationStatus is the quotation processing lifecycle (spec §4.1).
type QuotationStatus string

const (
	QuotationUploaded    QuotationStatus = "uploaded"
	QuotationProcessing  QuotationStatus = "processing"
	QuotationNeedsReview QuotationStatus = "needs_review"
	QuotationReady       QuotationStatus = "ready"
	QuotationFailed      QuotationStatus = "failed"
)

// ValidQuotationStatus reports whether a status value is known.
func ValidQuotationStatus(s QuotationStatus) bool {
	switch s {
	case QuotationUploaded, QuotationProcessing, QuotationNeedsReview, QuotationReady, QuotationFailed:
		return true
	default:
		return false
	}
}

// Allowed quotation upload extensions (ADR-008 allowlist).
var allowedQuotationExtensions = map[string]bool{
	".pdf":  true,
	".docx": true,
	".xlsx": true,
	".csv":  true,
}

// MaxQuotationUploadBytes is the upload size cap (spec §11).
const MaxQuotationUploadBytes = 10 << 20 // 10 MiB

// ValidateQuotationFilename checks the extension allowlist on the
// original client filename (ADR-008). Content sniffing happens at
// upload time in the service on the actual bytes.
func ValidateQuotationFilename(filename string) error {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" {
		return fmt.Errorf("filename is required")
	}
	if len(trimmed) > 255 {
		return fmt.Errorf("filename must be at most 255 characters")
	}
	ext := strings.ToLower(filepath.Ext(trimmed))
	if !allowedQuotationExtensions[ext] {
		return fmt.Errorf("file type not supported (allowed: .pdf, .docx, .xlsx, .csv)")
	}
	return nil
}

// ValidateVendorName checks a vendor display name after trimming.
func ValidateVendorName(name string) error {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return fmt.Errorf("vendor name is required")
	case len(trimmed) > 200:
		return fmt.Errorf("vendor name must be at most 200 characters")
	default:
		return nil
	}
}

// ProcessingJob is a row in the PostgreSQL-backed job queue (ADR-004):
// one extraction attempt per quotation upload.
type ProcessingJob struct {
	ID           string
	QuotationID  string
	Status       JobStatus
	Attempts     int
	MaxAttempts  int
	Kind         string // "extract" in the MVP
	ErrorMessage string
	FinishedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// JobStatus is the processing-job lifecycle.
type JobStatus string

const (
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobCompleted JobStatus = "completed"
	JobFailed    JobStatus = "failed"
)

// ValidJobStatus reports whether a status value is known.
func ValidJobStatus(s JobStatus) bool {
	switch s {
	case JobPending, JobRunning, JobCompleted, JobFailed:
		return true
	default:
		return false
	}
}