-- Cada NFS-e gera no máximo um título vivo no Contas a Receber (idempotência da emissão).
CREATE UNIQUE INDEX IF NOT EXISTS uq_titulos_nfse
    ON financeiro.titulos_receber(nfse_id)
    WHERE nfse_id IS NOT NULL AND status <> 'cancelado';
