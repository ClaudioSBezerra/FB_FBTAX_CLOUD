---
plan: 01-02
status: complete
completed_at: "2026-05-20"
commit: 8116eda
checkpoint: aprovado
---

# Summary: 01-02 — EmpresaPage React + rota App.tsx

## What was built

- `frontend/src/pages/EmpresaPage.tsx` — Página com dois formulários independentes (Dados da Empresa + Dados Bancários). Estado controlado simples (sem react-hook-form). Formulário bancário desabilitado até empresa ser salva. `useEffect` carrega dados existentes via GET no mount. Monkey-patch do AuthContext injeta headers de auth automaticamente.
- `frontend/src/App.tsx` — Import e rota `/admin/financeiro/empresa` registrada dentro de `ProtectedRoute`, entre a rota `/admin` e o fallback `*`.

## Verification

- `npm run build` → exit 0 (sem erros TypeScript)
- Checkpoint humano aprovado: redirect sem auth, dois cards, salvar empresa desbloqueia bancário, reload persiste dados, edição isolada entre formulários

## Decisions made

- `toast` importado de `sonner` (via alias `@/components/ui/sonner` não necessário — importar direto de `sonner`)
- Sem react-hook-form conforme especificação da fase 1
- `disabled={!empresa.id}` aplicado a todos os inputs e ao botão do formulário bancário
