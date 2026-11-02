-- O Id do DPS (DPS + município + CNPJ + série + número) não carrega o ambiente: o DPS 101 de
-- produção tem o mesmo Id do DPS 101 de produção restrita. A unicidade global de id_dps
-- impedia emitir em produção números já usados em teste. A unicidade correta é por ambiente,
-- já garantida por uq_nfse_emitidas_dps (empresa_id, ambiente, serie, n_dps).
ALTER TABLE financeiro.nfse_emitidas DROP CONSTRAINT IF EXISTS nfse_emitidas_id_dps_key;
