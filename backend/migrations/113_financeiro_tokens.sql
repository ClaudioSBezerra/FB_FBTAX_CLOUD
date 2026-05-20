CREATE TABLE IF NOT EXISTS financeiro.tokens (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id    UUID NOT NULL REFERENCES financeiro.contratos(id) ON DELETE CASCADE,
    plano_id       UUID NOT NULL REFERENCES financeiro.planos(id),
    token          VARCHAR(64) NOT NULL UNIQUE,
    status         VARCHAR(20) NOT NULL DEFAULT 'ativo',
    valid_from     DATE NOT NULL,
    valid_until    DATE NOT NULL,
    alerta_enviado BOOLEAN NOT NULL DEFAULT false,
    predecessor_id UUID REFERENCES financeiro.tokens(id),
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tokens_contrato_id ON financeiro.tokens(contrato_id);
CREATE INDEX IF NOT EXISTS idx_tokens_status      ON financeiro.tokens(status);
CREATE INDEX IF NOT EXISTS idx_tokens_valid_until ON financeiro.tokens(valid_until);
CREATE INDEX IF NOT EXISTS idx_tokens_token       ON financeiro.tokens(token);
