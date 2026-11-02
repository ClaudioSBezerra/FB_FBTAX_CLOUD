# NFS-e Nacional — runbook de emissão

Emissão direta na Sefin Nacional (sem provedor), com o A1 da Fortes Bezerra
(CNPJ 38.149.716/0001-28). Tela: **Financeiro › NFS-e** (`/admin/financeiro/nfse`).

## Antes do deploy

1. Definir `FINANCEIRO_SECRET` no Coolify (qualquer texto longo e secreto). **Sem ele o
   sistema recusa guardar o certificado.** Não troque depois: o PFX guardado deixa de abrir.
2. A migration `125_nfse_emissao.sql` roda sozinha no boot.
3. Se a conexão TLS com a Sefin falhar por cadeia ICP-Brasil desconhecida, montar o PEM das
   CAs e definir `NFSE_TLS_CA_FILE=/caminho/ca.pem` (mesmo expediente do FB_APU02).

## Ordem recomendada na tela

1. **Empresa** (menu Empresa): confirmar razão social, CNPJ e endereço do emitente.
2. **Certificado A1**: enviar o `.pfx` e a senha. O sistema confere validade e se o CNPJ do
   certificado é o do prestador antes de aceitar.
3. **Parâmetros fiscais**: IBGE `5201405`, telefone, e-mail. Regime já vem como Simples Nacional
   (ME/EPP, tributos federais e municipal pelo SN). Deixar inscrição municipal em branco, como
   nas notas emitidas no portal.
4. **Cliente**: se faltar o IBGE, usar "Buscar pelo CEP".
5. **Teste em produção restrita** (ambiente padrão, sem efeito fiscal): gerar a prévia e emitir
   R$ 1,00 de verdade nesse ambiente. Se a Sefin rejeitar, a mensagem aparece na lista de notas.
6. **Produção**: *Ativar produção* → gerar prévia → digitar `EMITIR EM PRODUCAO` → emitir R$ 1,00.
7. **Cancelar** a nota de teste na própria lista (justificativa de 15 a 255 caracteres).

## Garantias do sistema

- A prévia valida e assina o DPS **sem** gastar número e **sem** falar com a Sefin.
- Produção exige a frase de confirmação, no cliente e no servidor (envio e cancelamento).
- Número de DPS: série 70000, ambiente separado; produção começa no 101 (1–8 foram usados no
  Emissor Web). Número rejeitado **não é reaproveitado**.
- Se a resposta da Sefin se perder (nota `enviada`), use **Consultar** antes de reemitir.
- Uma nota viva por contrato e competência (índice único).

## Limites conhecidos

- Perfil fixo: serviço 17.01.01 / NBS 114013900, ISSQN no DAS (Simples), IBS/CBS informativo
  (CST 000, cClassTrib 000001), local da prestação = município do tomador (como nas notas reais).
- CNPJ alfanumérico (2026) não é aceito pelo leiaute DPS 1.01 — o sistema recusa com mensagem.
- O formato JSON das respostas da Sefin é interpretado de forma tolerante e o corpo bruto fica
  em `nfse_emitidas.erro`; ajustar `ParseRespostaSefin` se a Sefin usar outros nomes de campo.
- Testes com banco: `make test-nfse-db` (banco descartável).
