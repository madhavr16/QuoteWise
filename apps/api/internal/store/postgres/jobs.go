package postgres

import (
	"context"
	"time"

	"github.com/madhavr16/QuoteWise/apps/api/internal/domain"
)

// This file implements the service layer's ProcessingJobStore port:
// the PostgreSQL-backed job queue with FOR UPDATE SKIP LOCKED
// claiming (ADR-004).

const jobColumns = "id, quotation_id, status, attempts, max_attempts, kind, error_message, finished_at, created_at, updated_at"

// CreateProcessingJob enqueues an extraction job for a quotation.
func (s *Store) CreateProcessingJob(ctx context.Context, job domain.ProcessingJob) (domain.ProcessingJob, error) {
	return s.scanJob(
		s.pool.QueryRow(ctx, `
			INSERT INTO processing_jobs (quotation_id, kind, max_attempts)
			VALUES ($1, $2, $3)
			RETURNING `+jobColumns,
			job.QuotationID, job.Kind, job.MaxAttempts,
		),
	)
}

// ClaimNextPendingJob atomically claims the oldest pending job and
// marks it running: FOR UPDATE SKIP LOCKED means concurrent workers
// never claim the same row (ADR-004). domain.ErrNotFound when the
// queue is empty.
func (s *Store) ClaimNextPendingJob(ctx context.Context) (domain.ProcessingJob, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ProcessingJob{}, err
	}
	// Rollback is a no-op after a successful commit.
	defer tx.Rollback(ctx)

	job, err := s.scanJob(
		tx.QueryRow(ctx, `
			SELECT `+jobColumns+`
			FROM processing_jobs
			WHERE status = 'pending'
			ORDER BY created_at, id
			FOR UPDATE SKIP LOCKED
			LIMIT 1`),
	)
	if err != nil {
		return domain.ProcessingJob{}, mapQueryError(err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE processing_jobs
		SET status = 'running', attempts = attempts + 1, updated_at = now()
		WHERE id = $1`,
		job.ID,
	)
	if err != nil {
		return domain.ProcessingJob{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ProcessingJob{}, err
	}
	job.Status = domain.JobRunning
	job.Attempts++
	return job, nil
}

// CompleteJob marks a job completed and records its finish time.
func (s *Store) CompleteJob(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE processing_jobs
		SET status = 'completed', finished_at = $1, updated_at = now()
		WHERE id = $2`,
		time.Now(), id,
	)
	return err
}

// FailJob marks a job failed with a sanitized error message (client
// messages stay sanitized; details live in server logs only).
func (s *Store) FailJob(ctx context.Context, id string, errorMessage string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE processing_jobs
		SET status = 'failed', error_message = $1, finished_at = $2, updated_at = now()
		WHERE id = $3`,
		errorMessage, time.Now(), id,
	)
	return err
}

func (s *Store) scanJob(row pgxRow) (domain.ProcessingJob, error) {
	var job domain.ProcessingJob
	var finishedAt *time.Time
	err := row.Scan(
		&job.ID, &job.QuotationID, &job.Status, &job.Attempts,
		&job.MaxAttempts, &job.Kind, &job.ErrorMessage, &finishedAt,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return domain.ProcessingJob{}, mapQueryError(err)
	}
	job.FinishedAt = finishedAt
	return job, nil
}