CREATE TABLE IF NOT EXISTS financeiro.contratos (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id    UUID NOT NULL REFERENCES financeiro.clientes(id),
    data_inicio   DATE NOT NULL,
    periodicidade VARCHAR(20) NOT NULL DEFAULT 'mensal',
    valor_total   NUMERIC(12,2) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'ativo',
    observacoes   TEXT,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contratos_cliente_id ON financeiro.contratos(cliente_id);
CREATE INDEX IF NOT EXISTS idx_contratos_status ON financeiro.contratos(status);

CREATE TABLE IF NOT EXISTS financeiro.contrato_cnpjs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id UUID NOT NULL REFERENCES financeiro.contratos(id) ON DELETE CASCADE,
    cnpj_id     UUID NOT NULL REFERENCES financeiro.cliente_cnpjs(id),
    UNIQUE (contrato_id, cnpj_id)
);

CREATE INDEX IF NOT EXISTS idx_contrato_cnpjs_contrato_id ON financeiro.contrato_cnpjs(contrato_id);

CREATE TABLE IF NOT EXISTS financeiro.contrato_itens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contrato_id UUID NOT NULL REFERENCES financeiro.contratos(id) ON DELETE CASCADE,
    plano_id    UUID NOT NULL REFERENCES financeiro.planos(id),
    valor_item  NUMERIC(12,2),
    UNIQUE (contrato_id, plano_id)
);

CREATE INDEX IF NOT EXISTS idx_contrato_itens_contrato_id ON financeiro.contrato_itens(contrato_id);
