---
plan: 02-03
status: complete
completed_at: "2026-05-20"
commit: efe207e
---

# Summary: 02-03 — ContratosPage (Frontend)

## What was built

- `frontend/src/pages/ContratosPage.tsx` (578 linhas) — Página com dois modos controlados por estado (`view: 'lista' | 'novo'`):
  - **Lista:** filtro por cliente via select, tabela com Data Início, Periodicidade, Valor Total (formatado em BRL), Status (Badge com variante por status), botão Editar → Dialog de edição com campos periodicidade, valor_total, status e observacoes (PUT).
  - **Formulário multi-etapa (3 etapas):** Etapa 1 — select de cliente ativo → fetch GET `/api/financeiro/clientes?id=` para carregar CNPJs. Etapa 2 — checkboxes de CNPJs do grupo (is_principal destacado). Etapa 3 — `<details><summary>` por produto com checkboxes de planos + input de valor_item, campos data_inicio, periodicidade, valor_total, observacoes.
  - POST body: `{cliente_id, data_inicio, periodicidade, valor_total, observacoes, cnpj_ids: string[], itens: [{plano_id, valor_item: number|null}]}`.
  - Após 201: toast.success, volta para lista, recarrega contratos do cliente criado.
- `frontend/src/App.tsx` — import + rota `/admin/financeiro/contratos` dentro de `ProtectedRoute`.

## Verification

- `npm run build` → exit 0 (sem erros TypeScript)
- ContratosPage.tsx: 578 linhas (acima do mínimo 250)
- Padrão fetch nativo, sem TanStack Query, sem react-hook-form
- `useSearchParams` do react-router-dom para leitura de `?view=novo`

## Decisions made

- Uma única rota `/admin/financeiro/contratos` (lista + formulário) em vez de rota separada `/contratos/novo` — modo controlado por estado interno
- `<details><summary>` para seleção de produtos/planos: simples, sem estado extra de "aberto/fechado" por produto
- Dialog de edição separado do formulário de criação (são fluxos distintos)
- Validação parseFloat + NaN guard antes de todos os POSTs/PUTs numéricos
