-- Migration 100: Portal schema + pt_products table + seed
-- Creates the `portal` schema and the core products table for FBTAX_CLOUD

CREATE SCHEMA IF NOT EXISTS portal;

CREATE TABLE IF NOT EXISTS portal.pt_products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    icon_url        VARCHAR(500),
    destination_url VARCHAR(500),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pt_products_is_active ON portal.pt_products(is_active);

-- Seed: 4 FBTax products (icon_url and destination_url to be filled via admin panel in Story 4.3)
INSERT INTO portal.pt_products (name, description, icon_url, destination_url, is_active)
VALUES
    ('Apuração Assistida', 'Ferramenta de apuração fiscal assistida', '', '', true),
    ('Simulador Fiscal',   'Simulador de cenários fiscais',            '', '', true),
    ('SmartPick',          'Inteligência para picking logístico',      '', '', true),
    ('Farol',              'Monitoramento e alertas fiscais',           '', '', true)
ON CONFLICT DO NOTHING;
