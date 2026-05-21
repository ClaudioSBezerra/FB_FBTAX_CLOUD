ALTER TABLE financeiro.contas_financeiras
    ADD COLUMN IF NOT EXISTS provedor_config TEXT;   -- JSON criptografado (AES-256-GCM)

-- Índice para filtrar contas com provedor configurado
CREATE INDEX IF NOT EXISTS idx_contas_fin_provedor
    ON financeiro.contas_financeiras(provedor)
    WHERE provedor IS NOT NULL;
