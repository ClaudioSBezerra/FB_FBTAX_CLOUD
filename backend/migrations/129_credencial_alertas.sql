-- Controle dos alertas de vencimento de credenciais (certificado do Inter e A1 da NFS-e): cada marco
-- (30, 15, 7, 3, 1 e 0 dias) é avisado uma vez por validade; ao renovar a credencial a validade muda
-- e os avisos recomeçam.
CREATE TABLE IF NOT EXISTS financeiro.credencial_alertas (
    chave       TEXT NOT NULL,
    validade    DATE NOT NULL,
    marco       INTEGER NOT NULL,
    enviado_em  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chave, validade, marco)
);
