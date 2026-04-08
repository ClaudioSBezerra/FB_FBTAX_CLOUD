-- Migration 102: pt_tenant_products table
-- Maps which products are contracted by each tenant

CREATE TABLE IF NOT EXISTS portal.pt_tenant_products (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES portal.pt_tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES portal.pt_products(id) ON DELETE CASCADE,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, product_id)
);

CREATE INDEX IF NOT EXISTS idx_pt_tenant_products_tenant ON portal.pt_tenant_products(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pt_tenant_products_product ON portal.pt_tenant_products(product_id);
