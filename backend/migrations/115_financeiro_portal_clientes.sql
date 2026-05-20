CREATE TABLE IF NOT EXISTS financeiro.portal_clientes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id    UUID NOT NULL UNIQUE REFERENCES financeiro.clientes(id),
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    ativo         BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_portal_clientes_email      ON financeiro.portal_clientes(email);
CREATE INDEX IF NOT EXISTS idx_portal_clientes_cliente_id ON financeiro.portal_clientes(cliente_id);
