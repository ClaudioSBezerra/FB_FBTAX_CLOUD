-- Termo de Entrega de Projeto — documento de aceite assinado pelo cliente ao
-- final da implantação de um módulo (FB_SMARTPICK como primeiro caso de uso).
--
-- O documento segue o Padrão de Documentos FBTECH (docs/templates/PADRAO-DOCUMENTOS-FBTECH.md).
-- Os campos aqui persistidos são exatamente os que o módulo "abre em tela" para
-- preenchimento antes da impressão; o que ficar vazio é impresso como linha
-- pautada para preenchimento à mão no ato da assinatura.

CREATE SEQUENCE IF NOT EXISTS financeiro.entrega_seq START 1;

CREATE TABLE IF NOT EXISTS financeiro.entregas_projeto (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero        VARCHAR(30) NOT NULL UNIQUE,

    cliente_id    UUID REFERENCES financeiro.clientes(id) ON DELETE SET NULL,
    contrato_id   UUID REFERENCES financeiro.contratos(id) ON DELETE SET NULL,
    produto_id    UUID REFERENCES financeiro.produtos(id) ON DELETE SET NULL,

    versao        VARCHAR(10)  NOT NULL DEFAULT '1.0',
    titulo        VARCHAR(255) NOT NULL DEFAULT 'Termo de Entrega e Aceite de Projeto',
    projeto       VARCHAR(255),                    -- nome do projeto/implantação
    escopo_resumo TEXT,                            -- parágrafo de abertura do escopo

    data_inicio     DATE,                          -- início da implantação
    data_golive     DATE,                          -- entrada em produção
    data_entrega    DATE NOT NULL DEFAULT CURRENT_DATE,
    garantia_dias   INTEGER NOT NULL DEFAULT 30,   -- prazo de garantia pós go-live
    assistida_dias  INTEGER NOT NULL DEFAULT 14,   -- operação assistida

    ambiente      VARCHAR(120),                    -- ex.: "WMS Winthor - CD Goiânia"
    responsavel_tecnico   VARCHAR(255),
    responsavel_comercial VARCHAR(255),

    observacoes   TEXT,
    linhas_livres INTEGER NOT NULL DEFAULT 4,      -- nº de linhas pautadas p/ ressalvas

    status        VARCHAR(20) NOT NULL DEFAULT 'rascunho'
                  CHECK (status IN ('rascunho','emitido','assinado','cancelado')),

    assinado_data BYTEA,                           -- PDF assinado devolvido pelo cliente
    assinado_nome VARCHAR(255),
    assinado_em   TIMESTAMP WITH TIME ZONE,

    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_entregas_cliente  ON financeiro.entregas_projeto(cliente_id);
CREATE INDEX IF NOT EXISTS idx_entregas_produto  ON financeiro.entregas_projeto(produto_id);
CREATE INDEX IF NOT EXISTS idx_entregas_status   ON financeiro.entregas_projeto(status);

-- Itens entregues: as "linhas" que o usuário abre em tela para descrever o que
-- está sendo entregue e o respectivo veredito do aceite.
CREATE TABLE IF NOT EXISTS financeiro.entrega_itens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entrega_id  UUID NOT NULL REFERENCES financeiro.entregas_projeto(id) ON DELETE CASCADE,
    ordem       INTEGER NOT NULL DEFAULT 0,
    descricao   TEXT   NOT NULL,
    detalhe     TEXT,
    situacao    VARCHAR(30) NOT NULL DEFAULT 'entregue'
                CHECK (situacao IN ('entregue','parcial','pendente','nao_aplicavel')),
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_entrega_itens_entrega ON financeiro.entrega_itens(entrega_id, ordem);

-- Pendências e ressalvas registradas no aceite.
CREATE TABLE IF NOT EXISTS financeiro.entrega_pendencias (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entrega_id  UUID NOT NULL REFERENCES financeiro.entregas_projeto(id) ON DELETE CASCADE,
    ordem       INTEGER NOT NULL DEFAULT 0,
    descricao   TEXT   NOT NULL,
    responsavel VARCHAR(255),
    prazo       DATE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_entrega_pend_entrega ON financeiro.entrega_pendencias(entrega_id, ordem);

-- Signatários do termo. Nome/cargo em branco são impressos como linha pautada.
CREATE TABLE IF NOT EXISTS financeiro.entrega_signatarios (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entrega_id  UUID NOT NULL REFERENCES financeiro.entregas_projeto(id) ON DELETE CASCADE,
    ordem       INTEGER NOT NULL DEFAULT 0,
    papel       VARCHAR(120) NOT NULL,   -- ex.: "PELA CONTRATADA", "GESTOR DO CD"
    nome        VARCHAR(255),
    cargo       VARCHAR(255),
    organizacao VARCHAR(255),
    documento   VARCHAR(30),             -- CPF
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_entrega_sign_entrega ON financeiro.entrega_signatarios(entrega_id, ordem);
