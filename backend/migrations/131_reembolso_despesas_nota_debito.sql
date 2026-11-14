-- Reembolso de despesas de viagem: despesas (com comprovante) por cliente, fechadas em uma
-- Nota de Débito (modelo FOR.FIN.0002) que gera um título 'reembolso' no Contas a Receber.
-- Reembolso não é faturamento: não tem NFS-e e não renova token.
CREATE TABLE IF NOT EXISTS financeiro.despesas_comprovantes (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome         VARCHAR(255) NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    dados        BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS financeiro.notas_debito (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id    UUID NOT NULL REFERENCES financeiro.clientes(id) ON DELETE RESTRICT,
    codigo        VARCHAR(40) NOT NULL,            -- FOR.FIN.0002-MMAAAA[-n]
    emissao       DATE NOT NULL,
    vencimento    DATE NOT NULL,                   -- emissão + 10 dias
    destino       TEXT,
    periodo_ini   DATE NOT NULL,
    periodo_fim   DATE NOT NULL,
    solicitante   VARCHAR(120),
    total         NUMERIC(15,2) NOT NULL CHECK (total > 0),
    titulo_id     UUID REFERENCES financeiro.titulos_receber(id) ON DELETE SET NULL,
    xlsx          BYTEA NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'emitida' CHECK (status IN ('emitida', 'cancelada')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_notas_debito_codigo UNIQUE (codigo)
);

CREATE TABLE IF NOT EXISTS financeiro.despesas_viagem (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id      UUID NOT NULL REFERENCES financeiro.clientes(id) ON DELETE RESTRICT,
    data            DATE NOT NULL,
    tipo_despesa    VARCHAR(60) NOT NULL,          -- lista fixa do modelo (services.TiposDespesaViagem)
    descricao       TEXT NOT NULL,
    valor           NUMERIC(15,2) NOT NULL CHECK (valor > 0),
    comprovante_id  UUID REFERENCES financeiro.despesas_comprovantes(id) ON DELETE SET NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pendente' CHECK (status IN ('pendente', 'aprovada', 'descartada')),
    obs             TEXT,
    nota_debito_id  UUID REFERENCES financeiro.notas_debito(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_despesas_cliente_status ON financeiro.despesas_viagem(cliente_id, status);
CREATE INDEX IF NOT EXISTS idx_despesas_nota ON financeiro.despesas_viagem(nota_debito_id);

ALTER TABLE financeiro.titulos_receber DROP CONSTRAINT IF EXISTS titulos_receber_tipo_check;
ALTER TABLE financeiro.titulos_receber
    ADD CONSTRAINT titulos_receber_tipo_check CHECK (tipo IN ('mensalidade', 'adesao', 'projeto', 'outro', 'reembolso'));
