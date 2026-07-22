---
quick_id: 260722-n8p
slug: tema-dark-glass-green-portal-e-login
date: 2026-07-22
status: complete
commits:
  - cd6fb7f feat(theme): paleta dark glass-green e utilitários de glow
  - 7237135 feat(theme): ThemeProvider, toggle claro/escuro e anti-flash
  - 0503c3b feat(portal): aplica tema glass-green no portal público e nos logins
  - a80dc31 fix(theme): mantém telas não retemadas no claro sob o tema escuro global
---

# Quick Task 260722-n8p — Resumo

Tema dark "glass-green" (navy + teal, glassmorphism) aplicado ao portal público
e às telas de login, com toggle claro/escuro. Tema claro preservado byte a byte
na identidade vermelha Fortes Bezerra.

## O que foi feito

**Tokens** — bloco `.dark` em `frontend/src/index.css` reescrito com a paleta da
referência. `:root` inalterado.

**Utilitários** — `glow-teal`, `glow-card`, `glow-border`, `backdrop-glow`,
`hover-lift`, `hover-float`, `button-hover`, `text-glow`, `ambient-light(-2)` e
`accent-card`. Todos escopados em `.dark`; desligados sob `prefers-reduced-motion`.

**Tema** — `ThemeProvider` (next-themes) em `main.tsx` com `defaultTheme="dark"`,
componente `ThemeToggle` (sol/lua) no portal e no login, script anti-flash em
`index.html`.

**Telas** — `PortalPage`, `ProductCard`, `Footer`, `Login`, `PortalLoginPage`
migradas de cores fixas para tokens semânticos + pares `dark:`.

## Desvios do plano

**Guard de tema nas rotas fora do escopo (adicionado).** O plano tratava o tema
como se fosse escopável por rota. Não é: `defaultTheme="dark"` vale para o app
inteiro, e o módulo financeiro — cores claras fixas no markup + componentes
shadcn escuros — ficaria ilegível. `ThemeProvider` aninhado com `forcedTheme`
não resolve, porque next-themes 0.4 trata provider aninhado como no-op
(`useContext(L) ? children : <Theme/>`).

Solução escolhida pelo usuário entre três opções: `LightThemeScope`, um wrapper
que remove a classe `dark` enquanto as rotas `/admin*` e `/portal/dashboard`
estão montadas. Usa `MutationObserver` porque efeitos de filho rodam antes dos
do pai — sem ele o provider reaplicaria `dark` na montagem.

Custo aceito: pequeno flash escuro ao entrar no admin com o tema escuro ativo.

**`ProductCard` via custom properties.** O plano previa `useTheme()` como
alternativa; custom properties venceram por não exigir re-render nem tratar
hidratação.

## Verificação

- `npx tsc --noEmit` — limpo.
- `npm run build` — passa. O warning CSS `-: T;` é preexistente (confirmado com
  `git stash`), não veio destas mudanças.
- Validação visual com Chrome headless sobre o `dist`, com mock de
  `/api/portal/products`:
  - portal em dark — hero navy, orbes, cards com glow por acento, footer teal;
  - portal em light — idêntico ao original (hero azul, sombras sólidas);
  - `/admin/login` em dark — painel teal, botão legível;
  - `/admin` com tema global dark — renderiza claro, botão vermelho FB preservado
    (guard confirmado).

## Adendo — cursor com rastro (commit c4e2d0b)

Pedido do usuário depois da entrega inicial. Eu havia cortado o cursor da
referência por conta própria; foi reincorporado em `GlowCursor.tsx`.

Duas mudanças em relação ao original, por robustez: `transform` no lugar de
`left`/`top` (20 elementos por frame via propriedades de layout forçam reflow)
e delegação de eventos no lugar de `querySelectorAll` na montagem (o original
pressupõe página estática; aqui o conteúdo muda a cada render do React).

Ativa só em tema escuro + ponteiro fino + viewport md+, e respeita
`prefers-reduced-motion`.

Validado por CDP com `Input.dispatchMouseEvent` — screenshot confirma rastro
esticado em movimento e anel ampliado sobre card. Chrome headless reporta
`pointer: none` e o CDP não emula essa media feature, então o teste faz
monkey-patch de `matchMedia`; o código de produção não foi afetado.

## Não feito (fora do escopo)

- Módulo financeiro (`/admin/financeiro/*`) e painel FBTax seguem só no claro.
  Quando forem retemados, remover `LightThemeScope` de `App.tsx`.
- `PortalDashboardPage` também está sob o guard.
- Hover 3D (`rotateX`/`rotateY`) da referência: não portado — num app de gestão
  com grids longos, atrapalha a leitura.

## Deploy

Commits locais em `main`. **Nenhum push feito** — push em `main` dispara
auto-pull de deploy no Coolify.
