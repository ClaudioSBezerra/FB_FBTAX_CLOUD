---
title: 'NFS-e Nacional — base de emissão (DPS + assinatura, sem envio)'
type: 'feature'
created: '2026-11-02'
baseline_commit: 570feda3f646782cb892bcb3aeaff30dcd2178f3
status: 'done'
review_loop_iteration: 0
context:
  - '{project-root}/.planning/REQUIREMENTS.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** A Fortes Bezerra (CNPJ 38149716000128, Simples Nacional) emite NFS-e à mão no Emissor Web nacional; o módulo financeiro não tem certificado A1, parâmetros fiscais do prestador, IBGE do tomador nem gerador de DPS.

**Approach:** Etapa 1 da emissão via Sefin Nacional: migrations, cofre do A1 (AES-GCM), gerador do DPS v1.01 igual ao das notas reais, assinatura XMLDSig e empacotamento gzip+base64, tudo coberto por testes. Nenhuma chamada de rede à Sefin.

## Boundaries & Constraints

**Always:** Go stdlib + `net/http`, SQL cru, migrations numeradas (próxima: 125). PFX e senha só cifrados (`services.EncryptConfig`) e nunca devolvidos por rota. `nDPS` vem de contador transacional no banco, nunca reutilizado. Texto XML escapado. Id do DPS = `DPS`+cLocEmi(7)+tpInsc(1)+CNPJ(14)+série(5)+nDPS(15). Série padrão 70000.

**Ask First:** Nova dependência Go para XMLDSig/PKCS12 (`software.sslmate.com/src/go-pkcs12` e lib de c14n). Qualquer chamada real à Sefin.

**Never:** Enviar DPS à Sefin (produção ou restrita) nesta etapa. Emitir ou cancelar nota. Tocar nos 2 arquivos `frontend/package*.json` modificados. Calcular valores de IBS/CBS (a Sefin devolve).

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| DPS feliz | Prestador SN, tomador PE, R$ 24.000,00 | XML idêntico (campos) ao `<DPS>` da nota nº 1 e válido no XSD 1.01 | N/A |
| Valor R$ 1,00 | Tomador qualquer | DPS válido, `vServ` 1.00 | N/A |
| Tomador sem IBGE | `cMun` vazio | Erro antes de gerar | Erro claro, nada persistido |
| Nome com `&` | `FERREIRA COSTA & CIA` | `&amp;` no XML | N/A |
| Complemento com espaços repetidos | Vários espaços | Normalizado para um espaço | N/A |
| Assinatura | A1 de teste | `Signature` rsa-sha256 verificável com o certificado | Erro se PFX/senha inválidos |
| Contador concorrente | 2 geradores simultâneos | nDPS distintos e crescentes | Rollback não reaproveita número |

</frozen-after-approval>

## Code Map

- `backend/migrations/116_financeiro_empresa.sql`, `117_financeiro_clientes_endereco.sql` -- tabelas `financeiro.empresas`/`clientes` a estender.
- `backend/services/crypto.go` -- `EncryptConfig`/`DecryptConfig` (AES-256-GCM via FINANCEIRO_SECRET).
- `backend/handlers/financeiro.go:33` -- `EmpresaHandler`, onde expor os novos campos do prestador.
- `backend/handlers/cep.go`, `clientes.go` -- origem do endereço do tomador (sem IBGE hoje).
- FB_APU02 `nfse_pkcs12.go`, `nfse_credentials.go`, `nfse_adn.go` -- referência de decodificação de PFX e mTLS (copiar, não importar).
- `_bmad-output/...nfse-docs/xsd/Schemas/1.01/DPS_v1.01.xsd` -- validação; XML real da nota nº 1 como fixture.

## Tasks & Acceptance

**Execution:**
- [x] `backend/migrations/125_nfse_emissao.sql` -- colunas do prestador (IBGE, telefone, e-mail, IM, opSimpNac, regApTribSN, série, ambiente), `cod_ibge` no cliente, tabelas `financeiro.nfse_certificado`, `nfse_dps_contador` e `nfse_emitidas` -- base de dados
- [x] `backend/services/nfse_dps.go` -- structs e gerador do DPS 1.01 + Id + gzip/base64 -- espelha o DPS real
- [x] `backend/services/nfse_assinatura.go` -- carrega A1 do cofre e assina XMLDSig -- requisito da API
- [x] `backend/services/nfse_contador.go` -- reserva transacional de nDPS -- evita reuso
- [x] `backend/services/nfse_dps_test.go`, `nfse_assinatura_test.go` -- testes da matriz, fixture real e validação XSD

**Acceptance Criteria:**
- Given os dados da nota nº 1, when gero o DPS, then os elementos coincidem com o `<DPS>` do XML real e passam no XSD 1.01.
- Given um A1 de teste, when assino, then a assinatura verifica com a chave pública do certificado.
- Given dois pedidos simultâneos, when reservo nDPS, then os números são distintos.

## Design Notes

Algoritmo de assinatura e Reference (`#DPS...`, enveloped + exc-c14n) vêm do XSD/manual; confirmar na execução, pois a Sefin rejeitou com E999 mesmo com assinatura válida em relatos de terceiros. Validação XSD só em teste (lxml), sem dependência em produção.

## Verification

**Commands:**
- `cd backend && go build ./... && go vet ./services/... && go test ./services/ -run NFSe` -- expected: sem erros
- `python3 -I` com lxml no venv do scratchpad -- XML gerado válido no `DPS_v1.01.xsd`
