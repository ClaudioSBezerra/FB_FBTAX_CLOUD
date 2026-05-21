CREATE TABLE IF NOT EXISTS financeiro.inter_sync_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conta_id    UUID REFERENCES financeiro.contas_financeiras(id) ON DELETE SET NULL,
    iniciado_em TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    encerrado_em TIMESTAMP WITH TIME ZONE,
    status      VARCHAR(20) NOT NULL DEFAULT 'running',  -- running | ok | erro
    data_inicio DATE,
    data_fim    DATE,
    tx_importadas   INTEGER DEFAULT 0,
    tx_duplicadas   INTEGER DEFAULT 0,
    saldo_final     NUMERIC(15,2),
    erro_detalhe    TEXT
);
