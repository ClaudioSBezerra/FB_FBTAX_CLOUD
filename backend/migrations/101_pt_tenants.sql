-- Migration 101: pt_tenants table
-- Stores tenant identities (distributors/clients accessing the portal)

CREATE TABLE IF NOT EXISTS portal.pt_tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       VARCHAR(100) NOT NULL UNIQUE,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pt_tenants_slug ON portal.pt_tenants(slug);
