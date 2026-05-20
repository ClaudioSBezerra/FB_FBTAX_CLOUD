CREATE TABLE IF NOT EXISTS financeiro.api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id UUID NOT NULL REFERENCES financeiro.contratos(id) ON DELETE CASCADE,
    api_key     VARCHAR(64) NOT NULL UNIQUE,
    descricao   VARCHAR(255),
    ativo       BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (contrato_id)
);

CREATE INDEX IF NOT EXISTS idx_api_keys_contrato_id ON financeiro.api_keys(contrato_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_api_key     ON financeiro.api_keys(api_key);
