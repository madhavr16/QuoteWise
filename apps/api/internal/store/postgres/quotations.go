package postgres

import (
	"context"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// This file implements the service layer's QuotationStore port
// (ADR-015): every query is organization- and request-scoped
// (spec §11).

const quotationColumns = "id, procurement_request_id, organization_id, vendor_name, storage_key, checksum_sha256, original_filename, status, created_at, updated_at"

// CreateQuotation inserts a new quotation row and returns the stored
// row (server-generated timestamps).
func (s *Store) CreateQuotation(ctx context.Context, quotation domain.Quotation) (domain.Quotation, error) {
	return s.scanQuotation(
		s.pool.QueryRow(ctx, `
			INSERT INTO quotations (procurement_request_id, organization_id, vendor_name, storage_key, checksum_sha256, original_filename, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING `+quotationColumns,
			quotation.ProcurementRequestID, quotation.OrganizationID, quotation.VendorName,
			quotation.StorageKey, quotation.ChecksumSHA256, quotation.OriginalFilename, domain.QuotationUploaded,
		),
	)
}

// QuotationsByRequest lists a request's quotations, newest first.
func (s *Store) QuotationsByRequest(ctx context.Context, requestID string) ([]domain.Quotation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+quotationColumns+`
		FROM quotations
		WHERE procurement_request_id = $1
		ORDER BY created_at DESC, id`,
		requestID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	quotations := make([]domain.Quotation, 0)
	for rows.Next() {
		quotation, err := s.scanQuotation(rows)
		if err != nil {
			return nil, err
		}
		quotations = append(quotations, quotation)
	}
	return quotations, rows.Err()
}

// QuotationByID returns one quotation. domain.ErrNotFound when absent.
func (s *Store) QuotationByID(ctx context.Context, id string) (domain.Quotation, error) {
	return s.scanQuotation(
		s.pool.QueryRow(ctx, `
			SELECT `+quotationColumns+`
			FROM quotations
			WHERE id = $1`,
			id,
		),
	)
}

func (s *Store) scanQuotation(row pgxRow) (domain.Quotation, error) {
	var quotation domain.Quotation
	err := row.Scan(
		&quotation.ID, &quotation.ProcurementRequestID, &quotation.OrganizationID,
		&quotation.VendorName, &quotation.StorageKey, &quotation.ChecksumSHA256,
		&quotation.OriginalFilename, &quotation.Status,
		&quotation.CreatedAt, &quotation.UpdatedAt,
	)
	if err != nil {
		return domain.Quotation{}, mapQueryError(err)
	}
	return quotation, nil
}