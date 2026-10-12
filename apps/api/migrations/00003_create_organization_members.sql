-- +goose Up
-- Membership links users to organizations with a role
-- ('owner' | 'member'). Registration creates the owner membership
-- (ADR-007); inviting additional members lands later (open decision P6).

CREATE TABLE organization_members (
    organization_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, user_id)
);

CREATE INDEX organization_members_user_id_idx ON organization_members (user_id);
