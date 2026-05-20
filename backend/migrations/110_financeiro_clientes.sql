CREATE TABLE IF NOT EXISTS financeiro.clientes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razao_social VARCHAR(255) NOT NULL,
    cnpj         VARCHAR(14) NOT NULL UNIQUE,
    email        VARCHAR(255),
    telefone     VARCHAR(20),
    responsavel  VARCHAR(255),
    ativo        BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clientes_razao_social_trgm ON financeiro.clientes USING GIN (razao_social gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_clientes_ativo ON financeiro.clientes(ativo);

CREATE TABLE IF NOT EXISTS financeiro.cliente_cnpjs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id   UUID NOT NULL REFERENCES financeiro.clientes(id) ON DELETE CASCADE,
    cnpj         VARCHAR(14) NOT NULL UNIQUE,
    descricao    VARCHAR(255),
    is_principal BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cliente_cnpjs_cliente_id ON financeiro.cliente_cnpjs(cliente_id);
CREATE INDEX IF NOT EXISTS idx_cliente_cnpjs_cnpj_trgm ON financeiro.cliente_cnpjs USING GIN (cnpj gin_trgm_ops);
