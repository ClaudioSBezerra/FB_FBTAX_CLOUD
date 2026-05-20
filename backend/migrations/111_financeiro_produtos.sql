CREATE TABLE IF NOT EXISTS financeiro.produtos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo     VARCHAR(50) NOT NULL UNIQUE,
    nome       VARCHAR(255) NOT NULL,
    descricao  TEXT,
    ativo      BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO financeiro.produtos (codigo, nome, descricao) VALUES
    ('FB_APU02',     'Apuração Assistida',     'Ferramenta de apuração fiscal assistida'),
    ('FB_APU04',     'Simulador Fiscal',        'Simulador de cenários fiscais'),
    ('FB_SMARTPICK', 'SmartPick',               'Inteligência para picking logístico'),
    ('FB_FAROL',     'Farol',                   'Monitoramento e alertas fiscais')
ON CONFLICT (codigo) DO NOTHING;

CREATE TABLE IF NOT EXISTS financeiro.planos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    produto_id UUID NOT NULL REFERENCES financeiro.produtos(id) ON DELETE CASCADE,
    nome       VARCHAR(50) NOT NULL,
    preco      NUMERIC(12,2),
    ativo      BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (produto_id, nome)
);

CREATE INDEX IF NOT EXISTS idx_planos_produto_id ON financeiro.planos(produto_id);

INSERT INTO financeiro.planos (produto_id, nome, preco)
SELECT id, unnest(ARRAY['Lite','Standard','Premium','Enterprise','Sob Demanda']), NULL
FROM financeiro.produtos
ON CONFLICT (produto_id, nome) DO NOTHING;
