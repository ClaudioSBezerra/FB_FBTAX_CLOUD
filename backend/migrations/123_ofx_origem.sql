-- Migration 123: adiciona coluna origem em financeiro.transacoes e índice único parcial em referencia_ext
-- Contexto: habilita ingestão via upload OFX com rastreabilidade de origem e dedup robusto.

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'financeiro'
          AND table_name   = 'transacoes'
          AND column_name  = 'origem'
    ) THEN
        ALTER TABLE financeiro.transacoes
            ADD COLUMN origem VARCHAR(20) DEFAULT 'api_sync';
    END IF;
END $$;

-- Índice único parcial: dois registros com o mesmo referencia_ext não-nulo são sempre duplicata.
-- NULLs ficam fora do índice para não bloquear transações manuais sem ID externo.
CREATE UNIQUE INDEX IF NOT EXISTS idx_transacoes_referencia_ext_uq
    ON financeiro.transacoes(referencia_ext)
    WHERE referencia_ext IS NOT NULL;
