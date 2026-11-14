-- Detalhe bruto de cada lançamento bancário (extrato enriquecido do Inter: pagador, endToEnd,
-- identificadores de Pix e boleto). Guardado como veio, para a conciliação com Contas a Receber
-- poder usar os campos reais sem depender de descrição de texto.
ALTER TABLE financeiro.transacoes
    ADD COLUMN IF NOT EXISTS detalhes JSONB;
