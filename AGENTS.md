# Agent Guide

Persistent instructions for AI coding agents working in this repository.

## Product Context

This is a **mobile-first expense tracker**. The primary target is **iOS,
installed as a PWA from Safari** and launched standalone from the home
screen. It must also **look good on desktop web** — there is a 600px
breakpoint in `web/app/src/index.css` that centers the app as a 480px-wide
"card" on larger viewports, so don't break that layout.

When designing or changing UI:

- **Mobile-first CSS**: design for narrow viewports first, scale up with
  `min-width` media queries (existing breakpoint: `600px`).
- **iOS Safari PWA quirks** — the app already handles these; preserve them:
  - Use `100dvh` (with a `100vh` fallback) for full-height layouts.
  - Respect safe areas with `env(safe-area-inset-*)` on top/bottom chrome.
  - Inputs need explicit `user-select: text` / `-webkit-user-select: text`
    because `body` disables selection for the chromeless feel.
  - Use `touch-action: manipulation` to suppress the 300 ms tap delay.
  - Tap targets ≥ 44×44 px; no hover-only affordances.
  - Set `-webkit-tap-highlight-color: transparent` and provide your own
    pressed state (see the global `button:active` transform).
- **Desktop polish**: above the 600 px breakpoint the app sits in a
  centered card with a subtle shadow on a darker linen background — keep
  edges/shadows clean, don't stretch full-width.
- **Visual style** is "Linen & Ink" — calm, paper-textured, warm neutrals.
  Reference `mockups/` and the existing screens before introducing new
  colors or motion.

## Run tests after every change

After **every** code change, run the relevant test suite(s) below before
considering the task complete. Fix any failures or lints you introduce.

### Go (backend) — `*.go`

```bash
golangci-lint run
go test ./...
```

`go test ./...` covers `internal/...` and `cmd/...`. The E2E suite under
`./e2e/...` is excluded by default because it builds the React bundle and
launches Playwright; run it explicitly when you change UI flows or API
contracts (see below).

### Frontend — `web/app/src/**`

```bash
cd web/app
npx tsc --noEmit          # typecheck
npm run lint              # eslint
npm run build             # full type-check + production build
```

`npm run build` runs `tsc -b && vite build` and is the most thorough check
— prefer it before finishing a frontend task. It also refreshes
`web/dist/`, which Go embeds.

### End-to-end — `e2e/**` or any cross-cutting change

```bash
# First time only — versions MUST match playwright-go in go.mod
go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1
playwright install        # add --with-deps on Linux

go test -v ./e2e/...
```

The harness in `e2e/main_test.go` rebuilds `web/dist/` automatically when
it's missing or still the embed placeholder, then boots the server on
`:8081`. Selectors are anchored on `data-testid` — when adding UI that
e2e needs to reach, expose a stable `data-testid` rather than relying on
text or class names.

## Project layout

```
cmd/server      Go entry point, embeds web/dist
cmd/adduser     CLI to create users
internal/api    JSON handlers (auth, expenses, categories, insights)
internal/auth   Session middleware
internal/storage SQLite layer
internal/insights Pure stats/rollup logic
e2e             Playwright-driven end-to-end tests
web/app         Vite + React 19 + TS PWA source
web/dist        Built bundle (gitignored apart from a placeholder)
```

Stack: Go 1.25, SQLite via `modernc.org/sqlite` (CGo-free), React 19,
Vite 7, TanStack Query, `vite-plugin-pwa` + Workbox (read-only offline
via runtime caching of `GET /api/expenses`).

## House rules

- Don't commit `web/app/dev-dist/` (Vite PWA dev artifacts) or
  `expenses.db`.
- Don't introduce new top-level dependencies casually — this repo vendors
  Go deps and pins frontend deps; prefer extending what's already there.
- Keep components small; colocate `data-testid` attributes with the
  interactive element the test needs to click/read.
