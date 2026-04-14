# Story 2.3: Lógica de Tenant e Navegação

Status: ready-for-dev

## Story

Como cliente ativo do portal,
Quero que meus produtos contratados sejam destacados e clicáveis para acesso direto,
E que produtos não contratados exibam um CTA comercial discreto.

## Acceptance Criteria

**AC1:** Quando a URL contém `?tenant=slug`, a `PortalPage` lê o parâmetro via `useSearchParams()` do react-router-dom e passa `?tenant=slug` na chamada `GET /api/portal/products?tenant=slug` — sem chamada de API adicional (NFR16).

**AC2:** Com tenant na URL — produtos com `contracted: true` são exibidos com estilo ativo (borda colorida, sem opacidade reduzida) e são links clicáveis para `destination_url`, abrindo em nova aba com `target="_blank" rel="noopener noreferrer"` (NFR15).

**AC3:** Com tenant na URL — produtos com `contracted: false` são exibidos com opacidade reduzida (`opacity-60`) e um CTA discreto "Saiba mais" (link ou botão sem destino real nesta fase — href="#").

**AC4:** Quando a URL **não contém** `?tenant=`, todos os produtos são exibidos sem distinção de estado ativo/inativo — nenhum produto recebe estilo de "contracted" (FR4). O `queryKey` deve incluir o tenant para refetch automático ao mudar a URL.

**AC5:** `npm run build` passa sem erros e bundle permanece < 200kb gzipped (NFR2).

## Tasks / Subtasks

- [ ] Task 1: Atualizar `PortalPage.tsx` — ler tenant da URL e passar para a query (AC: #1, #4)
  - [ ] 1.1: Importar `useSearchParams` de `react-router-dom`
  - [ ] 1.2: `const [searchParams] = useSearchParams()` + `const tenant = searchParams.get('tenant') ?? ''`
  - [ ] 1.3: Incluir tenant no `queryKey`: `['products', tenant]`
  - [ ] 1.4: Incluir tenant na queryFn: `fetch(\`/api/portal/products${tenant ? \`?tenant=\${tenant}\` : ''}\`)`
  - [ ] 1.5: Passar `hasTenant={!!tenant}` como prop para cada `ProductCard`

- [ ] Task 2: Atualizar `ProductCard.tsx` — estilos por estado contracted/tenant (AC: #2, #3, #4)
  - [ ] 2.1: Adicionar prop `hasTenant: boolean` à interface `ProductCardProps`
  - [ ] 2.2: Produto `contracted: true` → sem opacidade, borda azul (`border-blue-500`), wrapper como `<a href={destination_url} target="_blank" rel="noopener noreferrer">`
  - [ ] 2.3: Produto `contracted: false` com tenant → `opacity-60`, badge "Não contratado" e link "Saiba mais" (href="#")
  - [ ] 2.4: Sem tenant (`hasTenant: false`) → estilo neutro igual para todos (sem opacidade, sem badge)

- [ ] Task 3: Validar (AC: #5)
  - [ ] 3.1: `npm run build` passa sem erros
  - [ ] 3.2: Bundle gzip < 200kb

## Dev Notes

### Leitura do ?tenant= sem API extra (NFR16)

```tsx
// PortalPage.tsx
import { useSearchParams } from 'react-router-dom'

const [searchParams] = useSearchParams()
const tenant = searchParams.get('tenant') ?? ''

const { data, isPending, isError } = useQuery<Product[]>({
  queryKey: ['products', tenant],
  queryFn: () =>
    fetch(`/api/portal/products${tenant ? `?tenant=${tenant}` : ''}`)
      .then(r => r.json()),
})
```

O endpoint `GET /api/portal/products?tenant=slug` já retorna `contracted: true/false` correto (Story 2.1). Nenhuma chamada extra necessária.

### Diferenciação visual por estado

| Situação | Estilo |
|---|---|
| Sem `?tenant=` na URL | Card neutro, sem distinção |
| `contracted: true` | Borda azul, sem opacidade, link clicável |
| `contracted: false` (com tenant) | `opacity-60`, badge, CTA "Saiba mais" |

### Wrapper de link para produto contratado (AC2)

```tsx
// Produto contratado com tenant — wrapper <a>
<a
  href={destination_url || '#'}
  target="_blank"
  rel="noopener noreferrer"
  className="block h-full"
>
  <Card className="border-blue-500 border-2 h-full hover:shadow-md transition-shadow cursor-pointer">
    ...
  </Card>
</a>
```

### Produto não contratado (AC3)

```tsx
// Adicionar ao rodapé do card
<a href="#" className="text-xs text-blue-600 hover:underline mt-2 inline-block">
  Saiba mais →
</a>
```

### Caso sem tenant (AC4)

Quando `hasTenant = false`, renderizar o `Card` normalmente sem links, sem opacidade, sem badges — todos os produtos como "disponíveis para contratação" (apresentação comercial neutra).

### Arquivos a modificar (APENAS estes)

- `frontend/src/pages/PortalPage.tsx` — adicionar `useSearchParams` + passar `hasTenant`
- `frontend/src/components/ProductCard.tsx` — adicionar `hasTenant` prop + lógica visual

### Learnings das stories anteriores

- `useSearchParams` está disponível via `react-router-dom` (já instalado)
- TanStack Query v5: `queryKey` deve incluir todas as variáveis que afetam a query (tenant)
- `npm run build` a partir de `frontend/`
- Path alias `@/` = `src/`
- Shadcn/UI `Card` em `@/components/ui/card`

### Referências

- [Source: epics.md#Story-2.3] — ACs completos
- [Source: frontend/src/pages/PortalPage.tsx] — arquivo a modificar
- [Source: frontend/src/components/ProductCard.tsx] — arquivo a modificar
- [Source: backend/handlers/portal_products.go] — endpoint já suporta ?tenant=

## Dev Agent Record

### Agent Model Used

_a ser preenchido_

### Debug Log References

_a ser preenchido_

### Completion Notes List

_a ser preenchido_

### File List

_a ser preenchido_

### Change Log

_a ser preenchido_
