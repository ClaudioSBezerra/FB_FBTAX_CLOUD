CREATE TABLE IF NOT EXISTS financeiro.contas_financeiras (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    apelido      VARCHAR(100) NOT NULL,
    banco        VARCHAR(100) NOT NULL,
    agencia      VARCHAR(20),
    conta        VARCHAR(30),
    tipo         VARCHAR(30) NOT NULL DEFAULT 'corrente',
    provedor     VARCHAR(50),
    provedor_id  VARCHAR(255),
    saldo        NUMERIC(15,2) NOT NULL DEFAULT 0,
    ultima_sync  TIMESTAMP WITH TIME ZONE,
    ativa        BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS financeiro.transacoes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conta_id       UUID NOT NULL REFERENCES financeiro.contas_financeiras(id) ON DELETE CASCADE,
    data_transacao DATE NOT NULL,
    descricao      VARCHAR(500) NOT NULL,
    valor          NUMERIC(15,2) NOT NULL,
    tipo           VARCHAR(10) NOT NULL CHECK (tipo IN ('credito','debito')),
    categoria      VARCHAR(100),
    referencia_ext VARCHAR(255),
    conciliado     BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transacoes_conta_data ON financeiro.transacoes(conta_id, data_transacao DESC);
CREATE INDEX IF NOT EXISTS idx_transacoes_data ON financeiro.transacoes(data_transacao DESC);
