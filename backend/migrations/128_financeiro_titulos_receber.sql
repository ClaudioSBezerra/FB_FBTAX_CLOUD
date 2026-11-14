-- Contas a Receber: um título por cobrança (mensalidade, adesão/setup, projeto). O vencimento é
-- informado no cadastro do título. Sem multa/juros e sem desconto (decisão de 14/11/2026).
-- A baixa liga o título ao lançamento bancário (transacoes) e, para mensalidade, renova o token
-- do contrato (services.RenovarTokensContrato).
CREATE TABLE IF NOT EXISTS financeiro.titulos_receber (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id     UUID NOT NULL REFERENCES financeiro.clientes(id) ON DELETE RESTRICT,
    contrato_id    UUID REFERENCES financeiro.contratos(id) ON DELETE SET NULL,
    tipo           VARCHAR(20) NOT NULL CHECK (tipo IN ('mensalidade', 'adesao', 'projeto', 'outro')),
    competencia    DATE,                         -- 1º dia do mês de referência (mensalidade)
    descricao      TEXT NOT NULL,
    valor          NUMERIC(15,2) NOT NULL CHECK (valor > 0),
    vencimento     DATE NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'aberto' CHECK (status IN ('aberto', 'pago', 'cancelado')),
    nfse_id        UUID REFERENCES financeiro.nfse_emitidas(id) ON DELETE SET NULL,

    -- cobrança emitida no banco (etapa BolePix)
    cobranca_provedor        VARCHAR(20),
    cobranca_codigo          VARCHAR(100),
    cobranca_status          VARCHAR(30),
    cobranca_pix_copia_cola  TEXT,
    cobranca_linha_digitavel TEXT,

    -- baixa
    pago_em        DATE,
    valor_pago     NUMERIC(15,2),
    transacao_id   UUID REFERENCES financeiro.transacoes(id) ON DELETE SET NULL,
    baixa_origem   VARCHAR(20) CHECK (baixa_origem IN ('automatica', 'manual', 'cobranca')),
    baixa_obs      TEXT,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_titulo_pago_tem_data CHECK (status <> 'pago' OR (pago_em IS NOT NULL AND valor_pago IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS idx_titulos_status_venc ON financeiro.titulos_receber(status, vencimento);
CREATE INDEX IF NOT EXISTS idx_titulos_cliente     ON financeiro.titulos_receber(cliente_id);
CREATE INDEX IF NOT EXISTS idx_titulos_contrato    ON financeiro.titulos_receber(contrato_id);
-- Um lançamento bancário baixa no máximo um título.
CREATE UNIQUE INDEX IF NOT EXISTS uq_titulos_transacao ON financeiro.titulos_receber(transacao_id) WHERE transacao_id IS NOT NULL;
-- Uma mensalidade viva por contrato e competência.
CREATE UNIQUE INDEX IF NOT EXISTS uq_titulos_mensalidade
    ON financeiro.titulos_receber(contrato_id, competencia)
    WHERE tipo = 'mensalidade' AND status <> 'cancelado' AND contrato_id IS NOT NULL AND competencia IS NOT NULL;
