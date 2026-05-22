CREATE TABLE IF NOT EXISTS financeiro.faixas_preco (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    produto_id UUID REFERENCES financeiro.produtos(id) ON DELETE CASCADE, -- NULL = global (todos os produtos)
    mb_min     NUMERIC(10,2) NOT NULL DEFAULT 0,
    mb_max     NUMERIC(10,2),         -- NULL = sem limite superior (sob consulta)
    preco      NUMERIC(12,2),         -- NULL = sob consulta
    descricao  VARCHAR(100),          -- label exibido ao cliente
    ativo      BOOLEAN NOT NULL DEFAULT true,
    ordem      INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_faixas_preco_produto ON financeiro.faixas_preco(produto_id);

-- Faixas globais padrão (produto_id NULL = aplica a todos os produtos)
INSERT INTO financeiro.faixas_preco (mb_min, mb_max, preco, descricao, ordem) VALUES
    (0,   20,  500.00,   'Até 20 MB',                 1),
    (20,  100, 1500.00,  'De 20 MB até 100 MB',       2),
    (100, 200, 3500.00,  'De 100 MB até 200 MB',      3),
    (200, 400, 5000.00,  'De 200 MB até 400 MB',      4),
    (400, 800, 7500.00,  'Acima de 400 MB até 800 MB',5),
    (800, NULL, NULL,    'Acima de 800 MB',            6)
ON CONFLICT DO NOTHING;
