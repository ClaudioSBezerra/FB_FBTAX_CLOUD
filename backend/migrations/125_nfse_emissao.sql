-- Emissão de NFS-e pelo Sistema Nacional (Sefin Nacional) — base de dados.
--
-- Estende o prestador (financeiro.empresas) com os parâmetros fiscais exigidos
-- pelo DPS v1.01, o tomador (financeiro.clientes) com o código IBGE do município,
-- e cria o cofre do certificado A1, o contador de DPS e o registro das notas.
-- O PFX e a senha ficam SEMPRE cifrados (services.CifrarCertificadoA1) e nenhuma
-- rota de leitura devolve esses campos.

-- Parâmetros fiscais do prestador. Os defaults de regime refletem o Simples Nacional ME/EPP
-- da Fortes Bezerra e DEVEM ser revistos ao cadastrar outro emitente. O ambiente nasce em 2
-- (produção restrita): a passagem para produção (1) é uma decisão explícita.
ALTER TABLE financeiro.empresas
    ADD COLUMN IF NOT EXISTS cod_ibge            VARCHAR(7)  CHECK (cod_ibge ~ '^[0-9]{7}$'),
    ADD COLUMN IF NOT EXISTS telefone            VARCHAR(20),
    ADD COLUMN IF NOT EXISTS email               VARCHAR(255),
    ADD COLUMN IF NOT EXISTS inscricao_municipal VARCHAR(20),
    ADD COLUMN IF NOT EXISTS op_simp_nac         SMALLINT NOT NULL DEFAULT 3 CHECK (op_simp_nac BETWEEN 1 AND 3),
    ADD COLUMN IF NOT EXISTS reg_ap_trib_sn      SMALLINT NOT NULL DEFAULT 1 CHECK (reg_ap_trib_sn BETWEEN 1 AND 3),
    ADD COLUMN IF NOT EXISTS reg_esp_trib        SMALLINT NOT NULL DEFAULT 0 CHECK (reg_esp_trib BETWEEN 0 AND 9),
    ADD COLUMN IF NOT EXISTS nfse_ambiente       SMALLINT NOT NULL DEFAULT 2 CHECK (nfse_ambiente IN (1, 2));
-- nfse_ambiente: 1 = produção, 2 = produção restrita. op_simp_nac 3 = ME/EPP optante;
-- reg_ap_trib_sn 1 = tributos federais e municipal pelo Simples Nacional.

ALTER TABLE financeiro.clientes
    ADD COLUMN IF NOT EXISTS cod_ibge VARCHAR(7) CHECK (cod_ibge ~ '^[0-9]{7}$');

CREATE TABLE IF NOT EXISTS financeiro.nfse_certificado (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id        UUID NOT NULL UNIQUE REFERENCES financeiro.empresas(id) ON DELETE RESTRICT,
    certificado_pfx   TEXT NOT NULL,           -- .pfx em base64, cifrado (AES-256-GCM)
    certificado_senha TEXT NOT NULL,           -- senha do .pfx, cifrada
    nome_arquivo      TEXT,
    cnpj_titular      VARCHAR(14),             -- extraído do CN do certificado
    validade          DATE NOT NULL,           -- NotAfter do certificado (extraído do próprio .pfx)
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contador de DPS por empresa, ambiente e série. O número reservado nunca é devolvido:
-- notas rejeitadas também consomem número (a Sefin não aceita DPS repetido). O primeiro número
-- da série 70000 é o 101 (1 a 8 foram usados no Emissor Web); as demais séries começam do 1
-- — essa regra vive em services.ReservarNDPS.
CREATE TABLE IF NOT EXISTS financeiro.nfse_dps_contador (
    empresa_id UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE RESTRICT,
    ambiente   SMALLINT NOT NULL CHECK (ambiente IN (1, 2)),
    serie      VARCHAR(5) NOT NULL CHECK (serie ~ '^[0-9]{5}$'),
    ultimo     BIGINT NOT NULL CHECK (ultimo >= 0),
    PRIMARY KEY (empresa_id, ambiente, serie)
);

CREATE TABLE IF NOT EXISTS financeiro.nfse_emitidas (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id     UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE RESTRICT,
    cliente_id     UUID REFERENCES financeiro.clientes(id) ON DELETE SET NULL,
    contrato_id    UUID REFERENCES financeiro.contratos(id) ON DELETE SET NULL,

    ambiente       SMALLINT NOT NULL CHECK (ambiente IN (1, 2)),   -- tpAmb usado no DPS
    serie          VARCHAR(5) NOT NULL CHECK (serie ~ '^[0-9]{5}$'),
    n_dps          BIGINT NOT NULL CHECK (n_dps > 0),
    id_dps         VARCHAR(45) NOT NULL UNIQUE,
    competencia    DATE NOT NULL,
    valor          NUMERIC(15,2) NOT NULL CHECK (valor > 0),        -- reais; o código trabalha em centavos
    descricao      TEXT NOT NULL,

    status         VARCHAR(20) NOT NULL DEFAULT 'gerada'
        CHECK (status IN ('gerada', 'enviada', 'autorizada', 'rejeitada', 'cancelada')),
    chave_acesso   VARCHAR(50),
    n_nfse         BIGINT,
    dps_xml        TEXT,                       -- DPS assinado enviado (contém dados do tomador)
    nfse_xml       TEXT,
    cancel_xml     TEXT,
    erro           TEXT,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_nfse_emitidas_dps UNIQUE (empresa_id, ambiente, serie, n_dps),
    CONSTRAINT ck_nfse_autorizada_tem_chave CHECK (status NOT IN ('autorizada', 'cancelada') OR chave_acesso IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_nfse_emitidas_status  ON financeiro.nfse_emitidas(empresa_id, status);
CREATE INDEX IF NOT EXISTS idx_nfse_emitidas_cliente ON financeiro.nfse_emitidas(cliente_id);
CREATE INDEX IF NOT EXISTS idx_nfse_emitidas_contrato ON financeiro.nfse_emitidas(contrato_id, competencia);
CREATE UNIQUE INDEX IF NOT EXISTS uq_nfse_emitidas_chave ON financeiro.nfse_emitidas(chave_acesso) WHERE chave_acesso IS NOT NULL;
-- Uma nota viva por contrato e competência (rejeitada/cancelada libera nova emissão).
CREATE UNIQUE INDEX IF NOT EXISTS uq_nfse_emitidas_contrato_competencia
    ON financeiro.nfse_emitidas(contrato_id, competencia, ambiente)
    WHERE contrato_id IS NOT NULL AND status IN ('gerada', 'enviada', 'autorizada');
