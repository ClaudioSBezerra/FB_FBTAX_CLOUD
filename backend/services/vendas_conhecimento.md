# Base de conhecimento do Assistente FbTax (pré-venda)

> Fonte: a landing apuracaoassistida.ia.br. Mantenha alinhado com o site: o bot só sabe o que está aqui.

## Empresa e produto
- A **FBTechIA** é a empresa (site www.fbtechia.com, CNPJ 38.149.716/0001-28). O **FBTax** é o produto.
- O FBTax é uma **solução privada**. Não é um sistema da Receita Federal nem do Comitê Gestor do IBS.
- Promessa: o governo calcula primeiro, a responsabilidade continua sendo da empresa. O FBTax compara a conta do governo com a conta da empresa, **nota a nota**, e mostra a diferença ainda dentro do prazo de ajuste.

## Reforma tributária (o essencial)
- A **CBS** (federal) substitui PIS e COFINS. O **IBS** (estados e municípios) substitui ICMS e ISS.
- **Apuração assistida**: a apuração da CBS e do IBS que o governo monta a partir das notas fiscais eletrônicas e apresenta para a empresa conferir. A **Receita Federal** cuida da CBS e o **Comitê Gestor do IBS (CGIBS)** cuida do IBS. Conferir continua sendo responsabilidade da empresa.
- Calendário: **2026** é ano de teste (as notas já trazem CBS de 0,9% e IBS de 0,1%). **2027**: a CBS passa a valer de verdade, PIS/COFINS acabam e começa o **split payment**. **2029–2032**: ICMS e ISS diminuem aos poucos e o IBS cresce. **2033**: só o modelo novo (CBS, IBS e Imposto Seletivo).
- **Crédito**: na regra nova, o crédito da compra só nasce quando o fornecedor paga o imposto dele.
- **Período de ajuste**: cada mês tem um prazo curto para ajustar; depois o saldo fica definitivo.
- **Split payment**: no pagamento (cartão, Pix, boleto) a parte da CBS e do IBS vai direto para o governo. Começa em 2027.

## Problemas que o FBTax resolve
- Nota que falta do lado do governo, ou com outro valor, vira imposto pago a mais.
- Crédito que não chega porque o fornecedor ainda não pagou: saber quais estão esperando e de quem cobrar.
- Prazo de ajuste que acaba.
- Dados espalhados (ERP, prefeitura, Receita, Comitê do IBS), que planilha não junta todo mês.

## Como funciona
1. **Integra o ERP**: notas de entrada e saída direto do **SAP S/4HANA** ou de bases **Oracle**; também importa **XMLs**; NFS-e pelo Ambiente Nacional.
2. **Busca o governo**: todo dia captura os débitos e créditos que a Receita (CBS) e o Comitê Gestor (IBS) calcularam, com as credenciais da própria empresa.
3. **Compara nota a nota** pela chave de cada nota: o que está igual, o que diverge e o que só existe de um lado.
4. **Aponta e avisa**: diferenças do mês, as notas que explicam cada uma e os prazos.

## Funcionalidades
Apuração do mês (CBS e IBS lado a lado com o governo) · Comparativo nota a nota · Malha fina antecipada · Crédito e fornecedores · Fechamento por estimativa · Alertas e relatórios (PDF e Excel) · Grupos com várias empresas (cada raiz de CNPJ separada, filiais na matriz) · "Aguardando" quando o governo ainda não entregou, nunca zero · Segurança (usuários, login corporativo SSO, credenciais criptografadas, isolamento por empresa, LGPD).

## Prova
Em produção num **grande grupo varejista de materiais de construção**, integrado ao **SAP S/4HANA**, conferindo centenas de milhares de notas por mês contra a Receita Federal. **Nunca cite o nome do cliente.**

## Planos e preço
- Há planos por **porte** (pequenas, médias e grandes empresas), sempre com **projeto de implantação** e **mensalidade**. O valor depende do **porte**, do **ERP** e do **número de empresas do grupo**.
- **O assistente nunca informa valores, faixas, descontos ou estimativas de preço.** O especialista apresenta na demonstração.
- Argumento: um mês de imposto pago a mais por uma nota que faltou costuma custar mais que a mensalidade.

## Contato e demonstração
- Demonstração com um especialista da FBTechIA, combinada pelo WhatsApp ou e-mail (contato@fbtechia.com).
- Termos e privacidade: https://apuracaoassistida.ia.br/termos-e-privacidade.html
