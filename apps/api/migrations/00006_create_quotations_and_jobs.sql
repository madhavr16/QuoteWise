-- +goose Up
-- Quotations: vendor documents uploaded under a procurement request.
-- File bytes live in object storage; only the storage key and
-- sha256 checksum are persisted here (ADR-008).

CREATE TABLE quotations (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    procurement_request_id UUID NOT NULL REFERENCES procurement_requests (id) ON DELETE CASCADE,
    organization_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    vendor_name            TEXT NOT NULL CHECK (char_length(btrim(vendor_name)) BETWEEN 1 AND 200),
    storage_key            TEXT NOT NULL CHECK (char_length(storage_key) BETWEEN 3 AND 255),
    checksum_sha256        TEXT NOT NULL CHECK (char_length(checksum_sha256) = 64),
    original_filename      TEXT NOT NULL CHECK (char_length(original_filename) BETWEEN 1 AND 255),
    status                 TEXT NOT NULL DEFAULT 'uploaded' CHECK (status IN ('uploaded','processing','needs_review','ready','failed')),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX quotations_procurement_request_id_idx ON quotations (procurement_request_id, created_at DESC);
CREATE INDEX quotations_organization_id_idx ON quotations (organization_id);

-- Processing jobs: PostgreSQL-backed job queue (ADR-004). One
-- extraction job per quotation upload; workers claim with
-- FOR UPDATE SKIP LOCKED.

CREATE TABLE processing_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quotation_id  UUID NOT NULL REFERENCES quotations (id) ON DELETE CASCADE,
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','completed','failed')),
    attempts      INTEGER NOT NULL DEFAULT 0,
    max_attempts  INTEGER NOT NULL DEFAULT 3,
    kind          TEXT NOT NULL DEFAULT 'extract' CHECK (kind = 'extract'),
    error_message TEXT,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX processing_jobs_status_idx ON processing_jobs (status, created_at);
CREATE INDEX processing_jobs_quotation_id_idx ON processing_jobs (quotation_id);