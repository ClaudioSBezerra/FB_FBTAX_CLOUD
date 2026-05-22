-- Sequência para número amigável do contrato (FB-2026-001)
CREATE SEQUENCE IF NOT EXISTS financeiro.contrato_seq START 1;

ALTER TABLE financeiro.contratos
    ADD COLUMN IF NOT EXISTS numero         VARCHAR(30),
    ADD COLUMN IF NOT EXISTS assinado_data  BYTEA,           -- PDF assinado (binário)
    ADD COLUMN IF NOT EXISTS assinado_nome  VARCHAR(255),    -- nome original do arquivo
    ADD COLUMN IF NOT EXISTS assinado_em    TIMESTAMP WITH TIME ZONE;

-- Preenche número para contratos já existentes
UPDATE financeiro.contratos
SET numero = 'FB-' || TO_CHAR(created_at, 'YYYY') || '-' || LPAD(NEXTVAL('financeiro.contrato_seq')::TEXT, 3, '0')
WHERE numero IS NULL;
