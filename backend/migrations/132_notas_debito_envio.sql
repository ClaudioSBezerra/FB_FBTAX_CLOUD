-- Registro do envio da nota de débito por e-mail (reenvio permitido, com confirmação na tela).
ALTER TABLE financeiro.notas_debito
    ADD COLUMN IF NOT EXISTS enviada_em   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS enviada_para TEXT;
