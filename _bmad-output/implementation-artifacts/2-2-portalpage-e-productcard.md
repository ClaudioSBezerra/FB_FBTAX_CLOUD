# Story 2.2: PortalPage e ProductCard

Status: review

## Story

Como visitante do portal,
Quero ver os produtos em uma grade visual com ícone, nome e descrição,
Para identificar rapidamente o portfólio da plataforma fbtax.cloud.

## Acceptance Criteria

**AC1:** A rota `/` em `App.tsx` renderiza `<PortalPage />` (não mais o placeholder `div`).

**AC2:** `PortalPage` usa `useQuery` do TanStack Query v5 para buscar `GET /api/portal/products` e exibe uma grade de `ProductCard` com os produtos retornados.

**AC3:** `ProductCard` exibe: ícone (SVG placeholder por enquanto — `icon_url` está vazio nesta fase), `name` e `description` de cada produto.

**AC4:** Durante o carregamento, exibe um estado de loading (skeleton ou spinner) — sem flash de conteúdo vazio.

**AC5:** Se a API falhar, exibe mensagem de erro amigável (sem stack trace).

**AC6:** A página é acessível sem autenticação (FR28) — nenhum redirect para `/admin/login`.

**AC7:** `npm run build` passa sem erros e o bundle permanece < 200kb gzipped (NFR2).

## Tasks / Subtasks

- [x] Task 1: Criar `frontend/src/components/ProductCard.tsx` (AC: #3)
  - [x] 1.1: Props: `id`, `name`, `description`, `icon_url`, `destination_url`, `contracted`
  - [x] 1.2: Card com Shadcn/UI `Card` + ícone placeholder emoji (📦) com fallback para `img` quando icon_url presente
  - [x] 1.3: Estilo neutro — sem distinção contracted/não-contracted (Story 2.3)

- [x] Task 2: Criar `frontend/src/pages/PortalPage.tsx` (AC: #2, #4, #5)
  - [x] 2.1: `useQuery` TanStack v5 com `queryKey: ['products']` e fetch `/api/portal/products`
  - [x] 2.2: `isPending` → 4 Skeletons de card
  - [x] 2.3: `isError` → mensagem de erro amigável
  - [x] 2.4: Success → grid responsivo 1/2/4 colunas com `ProductCard`
  - [x] 2.5: Header "Portal fbtax.cloud" + subtítulo + footer com ano dinâmico

- [x] Task 3: Atualizar `frontend/src/App.tsx` (AC: #1, #6)
  - [x] 3.1: `import PortalPage` + rota `/` renderiza `<PortalPage />`

- [x] Task 4: Validar (AC: #7)
  - [x] 4.1: `npm run build` → ✓ built in 2.64s
  - [x] 4.2: Bundle gzip: 87.93 kB (< 200 kB NFR2 ✅)

## Dev Notes

### TanStack Query v5 — mudanças críticas

```tsx
// v5: usar isPending (não isLoading)
const { data, isPending, isError } = useQuery({
  queryKey: ['products'],
  queryFn: () => fetch('/api/portal/products').then(r => r.json()),
})
```

`QueryClientProvider` já está em `App.tsx` — não recrie.

### Shadcn/UI disponível

Usar `Card`, `CardContent`, `CardHeader`, `CardTitle` de `@/components/ui/card`. Já instalados, não adicionar dependências.

### Ícone placeholder para esta story

`icon_url` está vazio no banco nesta fase. Usar um SVG inline genérico ou emoji como fallback:

```tsx
{icon_url ? (
  <img src={icon_url} alt={name} className="w-12 h-12" />
) : (
  <div className="w-12 h-12 rounded-full bg-blue-100 flex items-center justify-center text-2xl">
    📦
  </div>
)}
```

Story 4.3 (admin) permitirá atualizar os ícones reais.

### Tipo ProductResponse (reutilizar do backend — definir no frontend)

```ts
export type Product = {
  id: string
  name: string
  description: string
  icon_url: string
  destination_url: string
  contracted: boolean
}
```

### Grid responsivo (Tailwind)

```tsx
<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
  {data.map(product => <ProductCard key={product.id} {...product} />)}
</div>
```

### Proxy Vite (dev) — sem CORS em desenvolvimento

`vite.config.ts` já tem proxy `/api → localhost:8083`. Em produção, nginx roteia `/api/` para o Go backend.

### Learnings das stories anteriores

- `frontend/src/components/ui/` — NÃO MODIFICAR (Shadcn base)
- `frontend/src/contexts/AuthContext.tsx` — NÃO MODIFICAR
- `npm run build` a partir de `frontend/`
- Path alias `@/` = `src/`

### Referências

- [Source: epics.md#Story-2.2] — ACs completos
- [Source: frontend/src/App.tsx] — routing atual (placeholder a substituir)
- [Source: frontend/src/components/ui/card.tsx] — Card components disponíveis
- [Source: backend/handlers/portal_products.go] — response shape: id, name, description, icon_url, destination_url, contracted

## Dev Agent Record

### Agent Model Used

Claude Sonnet 4.6 (claude-sonnet-4-6)

### Debug Log References

Sem issues. Build limpo na primeira tentativa.

### Completion Notes List

- AC1 ✅: App.tsx rota `/` → `<PortalPage />`
- AC2 ✅: TanStack Query v5 `useQuery` + `isPending`/`isError`/`data`
- AC3 ✅: ProductCard com ícone placeholder emoji, nome e descrição
- AC4 ✅: 4 Skeletons durante `isPending`
- AC5 ✅: Mensagem de erro amigável em `isError`
- AC6 ✅: Rota pública sem auth (sem redirect)
- AC7 ✅: `npm run build` OK, bundle 87.93 kB gzip (< 200 kB)

### File List

**Criados:**
- `frontend/src/components/ProductCard.tsx` — card visual com Shadcn/UI Card + ícone placeholder
- `frontend/src/pages/PortalPage.tsx` — página pública principal com grade de produtos

**Modificados:**
- `frontend/src/App.tsx` — rota `/` substituída por `<PortalPage />`

### Change Log

| Data | Alteração |
|---|---|
| 2026-04-09 | PortalPage e ProductCard criados; grade responsiva com TanStack Query v5; skeleton loading; bundle 87.93 kB gzip |
