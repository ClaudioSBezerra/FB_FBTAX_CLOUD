# Padrão de Documentos FBTECH

Padrão visual e estrutural de todo documento institucional emitido pela FBTECH —
propostas, termos, contratos e relatórios de projeto.

Foi derivado dos três modelos de referência adotados pela empresa e reescrito com
a identidade FBTECH & IA. Os PDFs de origem **não estão versionados** — são
documentos confidenciais de terceiros e este repositório é público; veja
`README.md` desta pasta:

| Arquivo de referência | Tipo | O que foi aproveitado |
|---|---|---|
| `PT 33309 … Fase II-v.4.0.pdf` | Proposta Técnica | Estrutura de seções numeradas, matriz de responsabilidade, blocos de premissas |
| `PC 33309 … Fase II-v.4.0.pdf` | Proposta Comercial | Carta-proposta, tabelas de investimento, condições de faturamento |
| `PTC_33309 … v.2.0.pdf` | Proposta de Licenciamento | Restrição de uso e cópia, blocos de manutenção e SLA |

O escopo padrão do Termo de Entrega do SmartPick vem de um documento nosso, não de
modelo externo: `Proposta_SmartPick_Farol_JC_assinado.pdf` (JC Distribuição,
assinada em 24/03/2026), também fora do versionamento. Os entregáveis
pré-carregados na tela são a transcrição da seção 2.2 daquela proposta.

A implementação de referência é `backend/services/fbdoc.go`. Nenhum documento novo
deve montar PDF direto no maroto: use o kit.

---

## 1. Identidade

### 1.1. Marca

A assinatura é `docs/templates/fbtech-logo-dark.svg` — quadrado teal com o
monograma `FB`, seguido do logotipo bicolor `FB` + `TECH & IA`.

Nos PDFs a marca **não é uma imagem**. É redesenhada em primitivas vetoriais pela
função `services.LogoFBTECH`, o que mantém nitidez em qualquer zoom e na
impressão, e dispensa rasterizador no build. A largura do trecho `FB` do logotipo
é calculada pelas métricas AFM da Helvetica para que `TECH & IA` encoste nele com
precisão.

> A marca antiga (`backend/handlers/logo-fb.png`, monograma FB em bege) não é
> mais usada por nenhum documento e não é embarcada no binário.

### 1.2. Paleta

| Papel | Token | Hex | Uso |
|---|---|---|---|
| Institucional | `CorMarca` | `#0B1620` | Faixa da capa, títulos, cabeçalho de tabela |
| Destaque | `CorDestaque` | `#14B8A6` | Filete do cabeçalho, numeração de seção, papéis de assinatura |
| Texto | `CorTexto` | `#222A35` | Corpo |
| Apoio | `CorApoio` | `#64748B` | Legendas, rodapé, rótulos |
| Filete | `CorFiaR` | `#CBD5E1` | Réguas e bordas |
| Zebra | `CorZebra` | `#F1F5F9` | Linha alternada de tabela |
| Campo | `CorCampo` | `#94A3B8` | Traço das linhas de preenchimento manual |

Os modelos de origem usavam `#00174C` como institucional. O papel equivalente na
FBTECH é do grafite-petróleo `#0B1620`, tirado do fundo da logo.

### 1.3. Tipografia

Os modelos de origem foram diagramados em **Calibri** e **Century Gothic** —
fontes proprietárias que não podem ser embarcadas no binário. O padrão adota
**Helvetica** (métrica Arial, presente em qualquer leitor de PDF), preservando a
hierarquia de corpos dos originais:

| Elemento | Corpo (pt) | Estilo |
|---|---|---|
| Título da capa | 26 | Bold, caixa alta |
| Subtítulo da capa | 13 | Normal, cor de apoio |
| Seção `1.` | 13 | Bold |
| Subseção `1.1.` | 11 | Bold |
| Sub-subseção `1.1.1.` | 9,5 | Bold |
| Corpo | 9,5 | Normal |
| Tabela | 8,5 | Normal |
| Legenda / rodapé | 7,5 | Normal |

Margens: 18 mm nas laterais, 12 mm no topo, 14 mm no pé.

**Alinhamento do corpo é à esquerda, não justificado.** É a única divergência
deliberada em relação aos modelos de origem: o maroto justifica também a última
linha do parágrafo, esticando-a de ponta a ponta sempre que o espaçamento
resultante fica abaixo de dez vezes o normal. O resultado seria irregular de
parágrafo para parágrafo.

---

## 2. Estrutura da página

### 2.1. Capa

Faixa escura de 34 mm com a marca em negativo → filete teal curto → título em
caixa alta → subtítulo → pares rótulo/valor (Cliente, Documento, Emissão, Local)
→ tarja de confidencialidade.

### 2.2. Cabeçalho (todas as páginas)

Razão social à esquerda, `<código do documento> · <data>` à direita, sobre um
filete teal de 0,6 mm.

### 2.3. Rodapé (todas as páginas)

Filete cinza, razão social · CNPJ · aviso de confidencialidade à esquerda;
`Página X de Y` à direita.

### 2.4. Código do documento

`<SIGLA> <NÚMERO> v.<VERSÃO>` — ex.: `TE 2026-001 v.1.0`. Repetido no cabeçalho
de todas as páginas, como nos modelos de origem.

| Sigla | Documento |
|---|---|
| `PT` | Proposta Técnica |
| `PC` | Proposta Comercial |
| `PL` | Proposta de Licenciamento |
| `TE` | Termo de Entrega e Aceite |
| `CT` | Contrato de Prestação de Serviços |

---

## 3. Kit de composição

```go
doc := services.NovoDoc(services.DocMeta{
    Sigla: "TE", Numero: "2026-001", Versao: "1.0",
    Titulo: "Termo de Entrega e Aceite de Projeto",
    Subtitulo: "Implantação FB_SMARTPICK",
    Cliente: "GRUPO JC DISTRIBUIDORA LTDA",
    Cidade: "Aparecida de Goiânia – GO", Data: time.Now(),
    Emitente: "FORTES BEZERRA TECNOLOGIA LTDA",
    EmitCNPJ: "38.149.716/0001-28", Confidencial: true,
})
doc.Capa()
doc.Secao("Identificação")
doc.Paragrafo("…")
doc.Tabela(colunas, linhas)
doc.BlocoAssinaturas(signatarios)
pdf, err := doc.Bytes()
```

| Método | Efeito |
|---|---|
| `Capa()` | Folha de rosto + quebra de página |
| `Secao` / `Subsecao` / `Subsubsecao` | Títulos com numeração automática `1.` / `1.1.` / `1.1.1.` |
| `Bloco` | Título com peso de seção, **sem** numeração automática — para documentos que já trazem a própria numeração no texto |
| `Paragrafo` / `Marcador` / `Nota` | Corpo, item de lista com bullet teal, observação em corpo reduzido |
| `Tabela` | Cabeçalho escuro + zebra; larguras em unidades do grid de 12 |
| `Campo` | Par rótulo/valor no corpo |
| `LinhaPreenchivel` | Rótulo + traço; imprime o valor sobre a linha quando informado |
| `LinhasEmBranco` | Linhas pautadas sem rótulo, para anotação manuscrita |
| `BlocoAssinaturas` | Signatários aos pares, com traço e campos de identificação |
| `LocalEData` | Fecho `Cidade, ____ de __________ de ______.` |
| `QuebraPagina` | Nova página sem deixar folha em branco |
| `ManterJunto` | Quebra a página se um bloco de altura N não couber — impede blocos partidos ao meio |

### 3.1. Regra do campo em branco

**Todo campo não informado vira linha pautada, nunca um espaço vazio.** É o que
permite emitir um termo parcialmente preenchido e completá-lo à mão na reunião de
assinatura. Vale para `LinhaPreenchivel`, para os campos de `BlocoAssinaturas` e
para as seções de escopo e pendências do Termo de Entrega.

### 3.2. Blocos que não podem ser partidos

`Secao`, `Subsecao` e `Bloco` chamam `ManterJunto` antes de emitir o título, de
modo que um título nunca fique órfão no pé da página. `BlocoAssinaturas` faz o
mesmo por par de signatários: papel, traço e campos de identificação precisam
ficar juntos, sob pena de o documento perder validade prática ao ser assinado.

Ao acrescentar um bloco novo ao kit, avalie se ele é indivisível e chame
`ManterJunto` com a altura estimada.

### 3.3. Rótulo do documento do signatário

`Signatario.Doc` é impresso com o rótulo `CPF` por padrão, o caso comum de
pessoa física. Quando quem assina a linha é a própria pessoa jurídica — como no
contrato, em que CONTRATANTE e CONTRATADA são empresas — informe
`RotuloDoc: "CNPJ"`.

---

## 4. Armadilhas do maroto (v2.4.0)

Três comportamentos custaram depuração e estão resolvidos dentro do kit. Quem for
estender `fbdoc.go` precisa conhecê-los:

1. **`row.WithStyle` apaga o fundo das colunas.** Em `row.Render`, as colunas são
   renderizadas com `createCell = (r.style == nil)`: se a linha tem estilo, o
   fundo das colunas deixa de ser pintado. Por isso a faixa da capa é pintada
   coluna a coluna — do contrário o quadrado teal do monograma sumiria.

2. **`FitlnCurrentPage` conta o cabeçalho duas vezes.** Ele soma `headerHeight` a
   `getRowsHeight(rows...)`, que já inclui as linhas do cabeçalho. `QuebraPagina`
   localiza o espaço restante por busca binária e compensa o viés com
   `Doc.altCab`. Um espaçador arbitrariamente alto (`row.New(1000)`) **não**
   serve: gera uma página inteiramente em branco.

3. **Células adjacentes preenchidas deixam emendas visíveis.** Quanto menos
   linhas e colunas pintadas, melhor: a faixa da capa é uma linha só, com duas
   colunas.

---

## 5. Documentos implementados

| Documento | Sigla | Handler | Rota |
|---|---|---|---|
| Termo de Entrega e Aceite | `TE` | `backend/handlers/entrega_projeto.go` | `/api/financeiro/entregas/pdf` |
| Contrato de Prestação de Serviços | `CT` | `backend/handlers/contrato_visualizar.go` | `/api/financeiro/contratos/pdf` |

Os dois documentos que a plataforma emite estão no padrão. Não há geração de PDF
fora do kit.

### 5.1. Notas da migração do contrato

O texto jurídico do contrato foi preservado **integralmente** — preâmbulo,
cláusulas 1ª a 9ª, disposições adicionais e encerramento. Mudou apenas a
apresentação: capa institucional, cabeçalho/rodapé do padrão, tabela de produtos
zebrada e blocos de assinatura do kit.

As cláusulas são abertas com `Bloco`, e não com `Secao`: elas já se identificam
como "CLÁUSULA 1ª" no próprio texto, e a numeração automática do kit duplicaria
essa marcação. Os subitens do texto ("3.1.", "7.2.") continuam sendo parte da
redação jurídica, não gerados pelo kit.

A marca antiga (`backend/handlers/logo-fb.png`) deixou de ser embarcada no
binário — `handlers/assets.go` foi removido junto com a função de tratamento de
imagem que a preparava para o cabeçalho. O arquivo PNG segue versionado no
repositório como registro histórico.
