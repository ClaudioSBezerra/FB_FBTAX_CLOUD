# docs/templates

Identidade e modelos de documento da FBTECH.

| Arquivo | O que é |
|---|---|
| `PADRAO-DOCUMENTOS-FBTECH.md` | O padrão de documentos da empresa — leia primeiro |
| `fbtech-logo-dark.svg` | Assinatura da marca, fonte do desenho vetorial nos PDFs |
| `acordo-confidencialidade.md` | Modelo de NDA |

## Os PDFs de referência não estão aqui

Este repositório é **público**. Os modelos que deram origem ao padrão e as
propostas assinadas ficam **fora do versionamento** — o `.gitignore` barra
`docs/templates/*.pdf`.

São eles:

| Arquivo | Por que não pode ser versionado |
|---|---|
| `PT 33309 … Fase II-v.4.0.pdf` | Proposta técnica da CAST/Pelissari, marcada confidencial, destinada à Ferreira Costa — documento de terceiro |
| `PC 33309 … Fase II-v.4.0.pdf` | Idem, proposta comercial |
| `PTC_33309 … v.2.0.pdf` | Idem, proposta de licenciamento |
| `Proposta_SmartPick_Farol_JC_assinado.pdf` | Proposta assinada da JC Distribuição: valores contratuais, CNPJ e endereço do cliente |

Guarde-os em `~/fbtax-docs-confidenciais/` e copie para cá quando precisar
consultá-los. Eles não são necessários para compilar nem para rodar nada — o
padrão está inteiramente descrito em `PADRAO-DOCUMENTOS-FBTECH.md`, e o escopo
padrão do SmartPick já está transcrito no preset da tela de Termos de Entrega.

**Nunca faça `git add -f` nesses arquivos.** Eles já estiveram no histórico uma
vez e precisaram ser removidos com `git-filter-repo`, o que exigiu reescrever
todos os commits e um force-push.
