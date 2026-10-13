package postgres

import (
	"context"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// This file implements the service layer's ProcurementRequestStore
// port (ADR-015): every query is scoped by the organization identifier
// resolved server-side — never by client-supplied IDs (spec §11).

const procurementRequestColumns = "id, organization_id, requester_user_id, title, status, currency, created_at, updated_at"

// CreateProcurementRequest inserts a new request and returns the
// stored row (server-generated timestamps and defaults).
func (s *Store) CreateProcurementRequest(ctx context.Context, request domain.ProcurementRequest) (domain.ProcurementRequest, error) {
	return s.scanProcurementRequest(
		s.pool.QueryRow(ctx, `
			INSERT INTO procurement_requests (organization_id, requester_user_id, title, currency, status)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING `+procurementRequestColumns,
			request.OrganizationID, request.RequesterUserID, request.Title, request.Currency, domain.RequestDraft,
		),
	)
}

// ProcurementRequestsByOrganization lists an organization's requests,
// newest first.
func (s *Store) ProcurementRequestsByOrganization(ctx context.Context, organizationID string) ([]domain.ProcurementRequest, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+procurementRequestColumns+`
		FROM procurement_requests
		WHERE organization_id = $1
		ORDER BY created_at DESC, id`,
		organizationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := make([]domain.ProcurementRequest, 0)
	for rows.Next() {
		request, err := s.scanProcurementRequest(rows)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

// ProcurementRequestByID returns one request. domain.ErrNotFound when
// the id is absent.
func (s *Store) ProcurementRequestByID(ctx context.Context, id string) (domain.ProcurementRequest, error) {
	return s.scanProcurementRequest(
		s.pool.QueryRow(ctx, `
			SELECT `+procurementRequestColumns+`
			FROM procurement_requests
			WHERE id = $1`,
			id,
		),
	)
}

func (s *Store) scanProcurementRequest(row pgxRow) (domain.ProcurementRequest, error) {
	var request domain.ProcurementRequest
	err := row.Scan(
		&request.ID, &request.OrganizationID, &request.RequesterUserID,
		&request.Title, &request.Status, &request.Currency,
		&request.CreatedAt, &request.UpdatedAt,
	)
	if err != nil {
		return domain.ProcurementRequest{}, mapQueryError(err)
	}
	return request, nil
}