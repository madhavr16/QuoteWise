-- +goose Up
-- Procurement requests: organization-scoped purchase requests.
-- One currency per request in the MVP (ADR-006).

CREATE TABLE procurement_requests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id   UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    requester_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title             TEXT NOT NULL CHECK (char_length(btrim(title)) BETWEEN 1 AND 200),
    status            TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','submitted','processing','ready','failed')),
    currency          TEXT NOT NULL CHECK (char_length(currency) BETWEEN 3 AND 10),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX procurement_requests_organization_id_idx ON procurement_requests (organization_id, created_at DESC);
CREATE INDEX procurement_requests_requester_user_id_idx ON procurement_requests (requester_user_id);