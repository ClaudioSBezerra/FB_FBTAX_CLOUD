-- 133: leads do Assistente FbTax (bot de vendas da landing apuracaoassistida.ia.br) — issue #1.
-- Uma linha por conversa (sessao). Os dados de contato só são gravados depois do consentimento
-- dado na própria conversa (consentimento_em). ip_hash: SHA-256 do IP, só para limites de abuso.
CREATE TABLE IF NOT EXISTS vendas_leads (
    id                 BIGSERIAL PRIMARY KEY,
    sessao             UUID NOT NULL UNIQUE,
    criado_em          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    atualizado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    nome               TEXT NOT NULL DEFAULT '',
    empresa            TEXT NOT NULL DEFAULT '',
    cargo              TEXT NOT NULL DEFAULT '',
    email              TEXT NOT NULL DEFAULT '',
    whatsapp           TEXT NOT NULL DEFAULT '',
    faixa_faturamento  TEXT NOT NULL DEFAULT '',
    erp                TEXT NOT NULL DEFAULT '',
    num_empresas       TEXT NOT NULL DEFAULT '',
    prazo              TEXT NOT NULL DEFAULT '',
    resumo             TEXT NOT NULL DEFAULT '',
    origem             TEXT NOT NULL DEFAULT '',   -- utm_source/utm_campaign/página
    status             VARCHAR(20) NOT NULL DEFAULT 'conversa', -- conversa | qualificado | quente | contatado | descartado
    consentimento_em   TIMESTAMPTZ,
    encaminhado_em     TIMESTAMPTZ,
    observacao         TEXT NOT NULL DEFAULT '',
    conversa           JSONB NOT NULL DEFAULT '[]'::jsonb,
    mensagens          INTEGER NOT NULL DEFAULT 0,
    ip_hash            TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_vendas_leads_status ON vendas_leads (status, atualizado_em DESC);
CREATE INDEX IF NOT EXISTS idx_vendas_leads_ip ON vendas_leads (ip_hash, criado_em);
