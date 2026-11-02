-- Emissão de NFS-e pelo Sistema Nacional (Sefin Nacional) — base de dados.
--
-- Estende o prestador (financeiro.empresas) com os parâmetros fiscais exigidos
-- pelo DPS v1.01, o tomador (financeiro.clientes) com o código IBGE do município,
-- e cria o cofre do certificado A1, o contador de DPS e o registro das notas.
-- O PFX e a senha ficam SEMPRE cifrados (services.EncryptConfig) e nenhuma rota
-- de leitura deve devolvê-los.

ALTER TABLE financeiro.empresas
    ADD COLUMN IF NOT EXISTS cod_ibge          VARCHAR(7),
    ADD COLUMN IF NOT EXISTS telefone          VARCHAR(20),
    ADD COLUMN IF NOT EXISTS email             VARCHAR(255),
    ADD COLUMN IF NOT EXISTS inscricao_municipal VARCHAR(20),
    ADD COLUMN IF NOT EXISTS op_simp_nac       SMALLINT NOT NULL DEFAULT 3,  -- 3 = ME/EPP optante do Simples Nacional
    ADD COLUMN IF NOT EXISTS reg_ap_trib_sn    SMALLINT NOT NULL DEFAULT 1,  -- 1 = tributos federais e municipal pelo SN
    ADD COLUMN IF NOT EXISTS reg_esp_trib      SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS nfse_ambiente     SMALLINT NOT NULL DEFAULT 2   -- 1 = produção, 2 = produção restrita
        CHECK (nfse_ambiente IN (1, 2));

ALTER TABLE financeiro.clientes
    ADD COLUMN IF NOT EXISTS cod_ibge VARCHAR(7);

CREATE TABLE IF NOT EXISTS financeiro.nfse_certificado (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id        UUID NOT NULL UNIQUE REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    certificado_pfx   TEXT NOT NULL,           -- .pfx em base64, cifrado (AES-256-GCM)
    certificado_senha TEXT NOT NULL,           -- senha do .pfx, cifrada
    nome_arquivo      TEXT,
    validade          DATE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contador de DPS por empresa e série. O número reservado nunca é devolvido:
-- notas rejeitadas também consomem número (a Sefin não aceita DPS repetido).
-- A série 70000 já foi usada no Emissor Web (DPS 1 a 8); o contador começa em 100.
CREATE TABLE IF NOT EXISTS financeiro.nfse_dps_contador (
    empresa_id UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    serie      VARCHAR(5) NOT NULL DEFAULT '70000',
    ultimo     BIGINT NOT NULL DEFAULT 100,
    PRIMARY KEY (empresa_id, serie)
);

CREATE TABLE IF NOT EXISTS financeiro.nfse_emitidas (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id     UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    cliente_id     UUID REFERENCES financeiro.clientes(id) ON DELETE SET NULL,
    contrato_id    UUID REFERENCES financeiro.contratos(id) ON DELETE SET NULL,

    ambiente       SMALLINT NOT NULL,          -- tpAmb usado no DPS
    serie          VARCHAR(5) NOT NULL,
    n_dps          BIGINT NOT NULL,
    id_dps         VARCHAR(45) NOT NULL,
    competencia    DATE NOT NULL,
    valor          NUMERIC(15,2) NOT NULL,
    descricao      TEXT NOT NULL,

    status         VARCHAR(20) NOT NULL DEFAULT 'gerada'
        CHECK (status IN ('gerada', 'enviada', 'autorizada', 'rejeitada', 'cancelada')),
    chave_acesso   VARCHAR(50),
    n_nfse         BIGINT,
    dps_xml        TEXT,
    nfse_xml       TEXT,
    erro           TEXT,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_nfse_emitidas_dps UNIQUE (empresa_id, serie, n_dps)
);

CREATE INDEX IF NOT EXISTS idx_nfse_emitidas_status ON financeiro.nfse_emitidas(empresa_id, status);
CREATE INDEX IF NOT EXISTS idx_nfse_emitidas_chave  ON financeiro.nfse_emitidas(chave_acesso);
