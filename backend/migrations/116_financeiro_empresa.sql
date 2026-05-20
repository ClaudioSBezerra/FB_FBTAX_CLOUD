CREATE SCHEMA IF NOT EXISTS financeiro;

CREATE TABLE IF NOT EXISTS financeiro.empresas (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    razao_social  VARCHAR(255) NOT NULL,
    nome_fantasia VARCHAR(255),
    cnpj          VARCHAR(18) NOT NULL,
    logradouro    VARCHAR(255) NOT NULL,
    numero        VARCHAR(20)  NOT NULL,
    complemento   VARCHAR(100),
    bairro        VARCHAR(100) NOT NULL,
    cep           VARCHAR(10)  NOT NULL,
    municipio     VARCHAR(100) NOT NULL,
    uf            CHAR(2)      NOT NULL,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS financeiro.dados_bancarios (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    empresa_id UUID NOT NULL REFERENCES financeiro.empresas(id) ON DELETE CASCADE,
    banco      VARCHAR(100) NOT NULL,
    agencia    VARCHAR(20)  NOT NULL,
    conta      VARCHAR(30)  NOT NULL,
    tipo_conta VARCHAR(20)  NOT NULL,
    titular    VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
