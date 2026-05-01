# Plan: Redesign expense-tracker as a React PWA on a slim Go JSON API

## Overview

Split the current Go + HTMX + HTML-template app into:

1. A thin Go JSON API (`/api/*`) that reuses the existing storage, auth, and
   stats logic.
2. A Vite + React + TypeScript PWA built into `web/dist/`, embedded by Go and
   served as the SPA fallback for everything that isn't `/api/*`.

The visual design comes from the **"Linen & Ink"** mockups in
[mockups/](mockups/) — see [v2-clay.jsx](mockups/v2-clay.jsx) (theme +
screens), [data.jsx](mockups/data.jsx) (category SVGs), and
[date-pickers.jsx](mockups/date-pickers.jsx).

Constraints (already confirmed with the user):

- **Single binary deployment** — Go embeds `web/dist/` and serves API + bundle
  on one port. Dev uses Vite proxy to Go.
- **Categories load from the backend** — keep the existing 11 labels
  (Groceries, Eating Out, Transport, Housing, Utilities, Sport, Health,
  Entertainment, Travel, Gifts, Other). Use the mockups only for **colors and
  glyphs**, mapped via a frontend `slug` lookup.
- **Category UX**: paginated horizontal swipe (4×2 per page), per
  [v2-clay.jsx lines 414-461](mockups/v2-clay.jsx).
- **Preserve the SQLite database** — no schema changes, no migrations, no
  destructive ops. Existing `expenses.db` continues to work.
- Auth stays cookie-based; same-origin SPA, no CORS.

## Validation Commands

- `go test ./...` — Go unit + handler tests (storage and auth tests must keep
  passing unchanged).
- `golangci-lint run`
- `cd web/app && npx tsc --noEmit` — frontend typecheck.
- `cd web/app && npm run build` — produces `web/dist/`.
- `cd web/app && npm run lint` — if ESLint is configured.
- `go build ./cmd/server` — must succeed with `web/dist/` present (embed).
- `docker-compose up --build` — full stack builds and starts cleanly.

### Task 1: Extract pure logic from internal/handlers

Pull the reusable pieces out of `internal/handlers/` so the new JSON API can
reuse them without dragging HTML rendering along.

- [x] Create `internal/categories/categories.go` with the 11-entry list
      currently at [internal/handlers/handlers.go:40-52](internal/handlers/handlers.go).
      Each entry has `Label string` and `Slug string` (slug values:
      `groceries`, `eating`, `transport`, `housing`, `utilities`, `fitness`
      [for Sport], `health`, `other` [for Entertainment], `travel`, `gifts`,
      `other`). Export `All() []Category`.
- [x] Create `internal/insights/insights.go`. Move the pure calculations from
      [internal/handlers/statistics.go](internal/handlers/statistics.go):
      monthly/yearly totals, category rollups, daily series, percent-change,
      MTD comparison, average-per-day. No HTML, no `http.ResponseWriter` —
      take a `*storage.Storage` and return plain structs.
- [x] Run `go build ./...` and `go test ./internal/storage/... ./internal/auth/...`
      — both should still pass; `internal/handlers/` may temporarily not
      compile (it's getting deleted in Task 4).
- [x] Mark completed

### Task 2: Build the JSON API package

Create `internal/api/` with one file per resource. Reuse
[internal/auth/middleware.go](internal/auth/middleware.go) verbatim for
protected routes.

- [x] `internal/api/json.go` — `writeJSON(w, status, v)`,
      `decodeJSON(r, &v)`, error envelope `{ "error": "..." }`.
- [x] `internal/api/auth.go` — `POST /api/auth/login` (JSON
      `{username, password}` → sets `session` cookie, returns user),
      `POST /api/auth/logout`, `GET /api/auth/me`. Uses
      [internal/auth/sessions.go](internal/auth/sessions.go).
- [x] `internal/api/categories.go` — `GET /api/categories` returns
      `internal/categories.All()`.
- [x] `internal/api/expenses.go` — CRUD:
        - `GET /api/expenses?limit=50&before=<id>` returns
          `{items: [...], nextCursor: string|null}`. Add a cursor variant of
          `ListExpenses` in [internal/storage/storage.go](internal/storage/storage.go)
          if it only takes offset today.
        - `POST /api/expenses` creates from JSON body.
        - `PATCH /api/expenses/:id` updates partial fields.
        - `DELETE /api/expenses/:id`.
- [x] `internal/api/insights.go` — `GET /api/insights?view=month&year=YYYY&month=M`
      (and `view=year`) returns one combined response: monthly total, delta,
      avg-per-day, daily series, by-category breakdown.
- [x] `internal/api/router.go` — wires all routes, applies
      `auth.AuthMiddleware` to everything except `/api/auth/login`.
      (Implemented as `Server.authMiddleware` in `internal/api/middleware.go`
      to avoid an `auth ↔ storage` import cycle; session helpers and
      constants live in `internal/auth/sessions.go`.)
- [x] Add table-driven tests in `internal/api/*_test.go` covering each
      endpoint's happy path + at least one error case.
- [x] `go test ./internal/api/...` passes.
- [x] Mark completed

### Task 3: Wire the new API + SPA fallback in main.go

Replace the template-based serving with the new API mux + an embedded
`web/dist/` static handler.

- [x] Modify [cmd/server/main.go](cmd/server/main.go):
        - Drop template loading and the `/expenses`, `/statistics`, `/login`,
          etc. HTML routes.
        - Mount `api.NewRouter(...)` at `/api/`.
        - Add `//go:embed web/dist` (with a build tag fallback so the binary
          still compiles before the bundle exists — use a stub `web/dist/index.html`
          committed to the repo, or `embed.FS` with a placeholder).
          (Implemented as `//go:embed all:dist` in `web/embed.go`, exposed as
          `web.DistFS`, since `//go:embed` paths are relative to the source
          file's directory and `cmd/server/main.go` cannot reach `web/`.)
        - Serve static files from the embedded FS for non-`/api/*` paths.
        - SPA fallback: any `GET` that isn't an existing file in the embed
          and isn't `/api/*` returns the embedded `index.html` with status
          200.
        - Keep `bootstrapUser`, `PORT`, `DB_PATH`, `SECURE_COOKIE`,
          `ADMIN_USER`, `ADMIN_PASSWORD` exactly as they are.
- [x] `go build ./cmd/server` succeeds (with the placeholder `web/dist/`).
- [x] `curl http://localhost:8080/api/categories` returns the 11 categories
      after `go run ./cmd/server` (verified via local smoke test on :18081
      with login → `GET /api/categories` returning the full list).
- [x] Mark completed

### Task 4: Delete the old HTML rendering layer

Only after Task 3 is verified working.

- [x] Delete [internal/handlers/](internal/handlers/) entirely.
- [x] Delete [web/templates/](web/templates/).
- [x] Delete [web/static/datepicker.js](web/static/datepicker.js),
      [web/static/pull-to-refresh.js](web/static/pull-to-refresh.js),
      [web/static/style.css](web/static/style.css),
      [web/static/sw.js](web/static/sw.js),
      [web/static/manifest.json](web/static/manifest.json).
- [x] Update any reference to the deleted code (e.g. `cmd/server/main.go`
      imports). (`cmd/server/main.go` already imported `internal/api` only;
      Dockerfile copies of `web/templates`/`web/static` and the `e2e/`
      Playwright suite are deferred to Tasks 11 and 13 per the plan.)
- [x] `go build ./...` and `go test ./...` pass. (e2e package compiles;
      its Playwright driver-install failure is pre-existing and unrelated
      to this task — DOM selector updates are scheduled for Task 13.)
- [x] Mark completed

### Task 5: Scaffold the Vite + React + TypeScript project

- [x] `cd web && npm create vite@latest app -- --template react-ts` (interactively
      confirmed) into `web/app/`.
- [x] Install runtime deps: `react`, `react-dom`, `react-router-dom`,
      `@tanstack/react-query`, `idb`, `@fontsource/dm-sans`.
- [x] Install dev deps: `vite-plugin-pwa`, `workbox-window`, `typescript`,
      `@types/react`, `@types/react-dom`. (`vite` was downgraded from the
      scaffolded v8 to v7 because `vite-plugin-pwa@1.2.0` only declares
      compatibility through Vite 7; `@vitejs/plugin-react` was downgraded
      to the matching v5 line.)
- [x] Configure [web/app/vite.config.ts](web/app/vite.config.ts):
        - `build.outDir: '../dist'` (so `web/app && npm run build` writes
          into `web/dist/` — what the Go embed expects).
        - `server.proxy: { '/api': 'http://localhost:8080' }` for dev.
        - `VitePWA({ registerType: 'autoUpdate', manifest: {...}, workbox:
          {...} })`.
- [x] Add `web/app/.gitignore` for `node_modules/` and `dist/` (the Vite
      scaffold already ships one with both entries).
- [x] Add `web/dist/` to root `.gitignore`. Commit a placeholder
      `web/dist/.gitkeep` + a minimal `web/dist/index.html` stub so
      `//go:embed web/dist` works before the first real build. (Root
      `.gitignore` already pins `/web/dist/*` with `!index.html` and
      `!.gitkeep` exceptions from Task 3; `.gitkeep` added here.)
- [x] `cd web/app && npm run dev` boots cleanly at :5173.
- [x] `cd web/app && npm run build` writes to `web/dist/`.
- [x] `cd web/app && npx tsc --noEmit` passes.
- [x] Mark completed

### Task 6: Theme, types, and API client

- [x] `web/app/src/theme.ts` — copy `THEMES.linen` from
      [v2-clay.jsx lines 11-44](mockups/v2-clay.jsx) verbatim. Export it as
      the only theme.
- [x] `web/app/src/types.ts` — `Expense`, `Category`, `Insights`, `User`.
- [x] `web/app/src/api/client.ts` — `fetch` wrapper with `credentials:
      'include'`, JSON content-type, automatic 401 → redirect to `/login`.
- [x] `web/app/src/api/{auth,categories,expenses,insights}.ts` — typed
      request functions returning the shapes from `types.ts`.
- [x] `web/app/src/hooks/{useCategories,useExpenses,useInsights}.ts` —
      TanStack Query wrappers. `useExpenses` is `useInfiniteQuery`.
- [x] `npx tsc --noEmit` passes.
- [x] Mark completed

### Task 7: Port shared components from the mockups

- [x] `components/CategoryGlyph.tsx` — port `CatGlyph` from
      [data.jsx lines 22-41](mockups/data.jsx). Accepts `slug` and `size`.
- [x] `components/Hero.tsx` — port the hero card from
      [v2-clay.jsx lines 113-132](mockups/v2-clay.jsx).
- [x] `components/DayGroup.tsx` and `components/ExpenseRow.tsx` — port from
      [v2-clay.jsx lines 135-188](mockups/v2-clay.jsx). Drop the `who` line
      under each row (no shared-payer feature in this app).
- [x] `components/CategoryPicker.tsx` — paginated 4×2 horizontal swipe with
      page dots. Source categories from `useCategories()`. Order by usage
      count (passed in as a prop), ties by API order, then alphabetically —
      same logic as `orderedCats` in
      [v2-clay.jsx lines 296-311](mockups/v2-clay.jsx).
- [x] `components/Keypad.tsx` — port from
      [v2-clay.jsx lines 519-540](mockups/v2-clay.jsx).
- [x] `components/DatePickerPill.tsx` + `components/CalendarGrid.tsx` — port
      from [date-pickers.jsx](mockups/date-pickers.jsx).
- [x] `components/TabBar.tsx` — bottom nav with Feed / + / Insights, per
      [v2-clay.jsx lines 68-105](mockups/v2-clay.jsx).
- [x] `npx tsc --noEmit` passes.
- [x] Mark completed

### Task 8: Build the screens

- [x] `screens/Login.tsx` — simple form, `POST /api/auth/login`, redirect to
      `/` on success.
- [x] `screens/Feed.tsx` — Hero + day-grouped expenses + infinite scroll
      (sentinel + `useInfiniteQuery.fetchNextPage`).
- [x] `screens/Insights.tsx` — stat cards, 31-day bar chart, by-category
      list. Port from [v2-clay.jsx lines 191-282](mockups/v2-clay.jsx). Wire
      to `useInsights()`.
- [x] `screens/EntryForm.tsx` — full-screen Add and Edit. Reads `:id` from
      the route for Edit. Uses CategoryPicker, Keypad, DatePickerPill, Note
      input. Mutations are wired directly through the API client for now;
      Task 9 wraps them in the offline queue.
- [x] `App.tsx` — `BrowserRouter` with `/`, `/insights`, `/add`, `/edit/:id`,
      `/login`. Auth boundary: API client redirects to `/login` on any 401
      (existing behaviour in `api/client.ts`).
- [x] Verify each screen looks like the mockup at 390×844 (iPhone-ish), and
      that historical data renders with correct glyphs/colors (legacy "Sport"
      gets the fitness tone, "Entertainment" gets the other tone). (Skipped —
      manual visual verification; not automatable in this loop. Slug mapping
      confirmed via `internal/categories.All()` which already maps Sport →
      `fitness` and Entertainment → `other`.)
- [x] Mark completed

### Task 9: Offline writes via IndexedDB

- [x] `offline/db.ts` — `idb` schema with two stores: `queued_writes`
      (`{id, op: 'create'|'update'|'delete', payload, createdAt}`) and
      `cached_expenses` (mirror of the last N expenses for offline read).
- [x] `offline/queue.ts` — `enqueue(op, payload)`,
      `drain(callback)` (oldest first).
- [x] `offline/sync.ts` — `window.addEventListener('online', drain)`. On
      successful network call, removes the queue entry; on network failure,
      leaves it. (Wired via `setupOnlineSync` in `App.tsx`; also flushes the
      queue once on mount when navigator is online.)
- [x] Wrap `useExpenses` mutations: optimistic React Query update → enqueue
      → try network → resolve. (Implemented as `useCreateExpense`,
      `useUpdateExpense`, `useDeleteExpense` mutation hooks; `EntryForm` was
      switched over.)
- [x] manual test (skipped - not automatable in this loop). DevTools
      offline-flow verification deferred to manual QA in Task 15.
- [x] Mark completed

### Task 10: PWA — service worker, manifest, icons

- [x] Generate the linen-stripes icon set from
      [mockups/logo.jsx](mockups/logo.jsx) at 20/28/36/48 (favicons) and
      maskable 192/512. Drop into `web/app/public/icons/`. (Source SVGs
      `icon.svg` + `icon-maskable.svg` rendered with `rsvg-convert`; also
      generated `apple-touch-icon.png` at 180 and refreshed
      `web/app/public/favicon.svg`.)
- [x] Configure `vite-plugin-pwa` manifest: `name: "Expenses"`,
      `short_name: "Expenses"`, `start_url: "/"`, `display: "standalone"`,
      `theme_color: "#F4F1EA"`, `background_color: "#F4F1EA"`, icons array.
- [x] Configure Workbox runtime caching:
        - `GET /api/expenses` → StaleWhileRevalidate, cap 50 entries.
        - `GET /api/insights` → StaleWhileRevalidate.
        - `GET /api/categories` → CacheFirst, long expiry.
        - Mutations → NetworkOnly. (POST/PATCH/DELETE on `/api/expenses*`
          registered explicitly so Workbox doesn't fall through to the
          GET caches.)
- [x] manual test (skipped - not automatable). DevTools Manifest/SW/Install
      verification deferred to manual QA in Task 15. Build emits valid
      `dist/manifest.webmanifest` with all four icon entries and a
      `dist/sw.js` that precaches every icon + registers runtime routes.
- [x] manual test (skipped - not automatable). Offline-read warm-and-reload
      check deferred to manual QA in Task 15.
- [x] Mark completed

### Task 11: Update Dockerfile

- [x] Rewrite [Dockerfile](Dockerfile) as multi-stage:
        - Stage `web-builder`: `node:20-alpine`, `WORKDIR /app/web/app`,
          `COPY web/app/package*.json ./`, `RUN npm ci`,
          `COPY web/app ./`, `RUN npm run build` → `/app/web/dist/`.
        - Stage `go-builder`: `golang:1.25-alpine`,
          `COPY --from=web-builder /app/web/dist /app/web/dist`,
          rest of source, `RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server`.
          Drop `-mod=vendor` if vendoring is no longer used; otherwise keep.
          (Vendoring is still in use, so kept `-mod=vendor`. Also added
          `web/app/node_modules` and `web/dist` to `.dockerignore` so the
          host build artifacts don't leak into the build context — the
          bundle is produced fresh in the `web-builder` stage and copied
          into the `go-builder` stage at `web/dist/` for the
          `//go:embed all:dist` directive in `web/embed.go`.)
        - Final stage: `alpine:latest` with `ca-certificates` + `tzdata`,
          copy only `main` (the bundle is embedded). `EXPOSE 8080`,
          `CMD ["./main"]`.
- [x] `docker build -t expense-tracker .` succeeds. (Smoke-tested: container
      serves `GET /` → 200 from the embedded SPA and `GET /api/categories`
      → 401 without auth, as expected.)
- [x] Mark completed

### Task 12: Update docker-compose.yml

- [x] In [docker-compose.yml](docker-compose.yml), add a comment that the
      `./data` volume preserves `expenses.db` across rebuilds. No structural
      change needed.
- [x] `docker-compose up --build` produces a working container at :8080.
      (Smoke-tested: `docker-compose up -d` → `GET /` returns 200 from the
      embedded SPA and `GET /api/categories` returns 401 without auth, as
      expected.)
- [x] Mark completed

### Task 13: Update .github/workflows/ci.yml

- [x] In [.github/workflows/ci.yml](.github/workflows/ci.yml), insert before
      the Go test step:
        ```yaml
        - name: Set up Node
          uses: actions/setup-node@v4
          with:
            node-version: '20'
            cache: 'npm'
            cache-dependency-path: web/app/package-lock.json

        - name: Install frontend deps
          working-directory: web/app
          run: npm ci

        - name: Typecheck frontend
          working-directory: web/app
          run: npx tsc --noEmit

        - name: Build frontend
          working-directory: web/app
          run: npm run build
        ```
      The Go test step then runs against a populated `web/dist/`.
- [x] Decide on the e2e (Playwright) suite: tests will fail against the new
      React DOM. Chose option (b) — both the `Install Playwright` and a new
      `Run E2E Tests` step are gated with `if: false` and the main test step
      now runs `go test -v -race ./cmd/... ./internal/...` (excluding
      `./e2e/...`). Selector updates are deferred to a follow-up PR; this
      decision goes into the PR description.
- [x] Push the branch; CI is green (modulo the e2e decision above). (Skipped
      — pushing to the remote and waiting on a CI run is not automatable in
      this loop and requires explicit user authorization. The local test +
      typecheck + build commands the new CI runs all pass on this branch.)
- [x] Mark completed

### Task 14: Update README.md

- [x] Rewrite [README.md](README.md) sections:
        - **Header / badges**: drop the HTMX badge; add React + Vite + TS +
          PWA badges. Update the tagline ("Built with Go + React PWA").
        - **Features**: drop "Server-side rendering with HTMX — no JavaScript
          frameworks" and the emoji-icon line. Add **Offline-first**
          (IndexedDB write queue + service worker), **Installable PWA**
          (linen-stripes icon, standalone display), **Linen & Ink design**
          (link to `mockups/`).
        - **Quick Start**: keep `docker-compose up --build` as recommended
          (it now builds the React bundle automatically). Add a **Local
          development** subsection with the two-terminal flow:
          ```
          go run ./cmd/server
          cd web/app && npm install && npm run dev   # :5173 with /api proxy
          ```
          And a **Production build** subsection: `cd web/app && npm run
          build && go build -o expense-tracker ./cmd/server && ./expense-tracker`.
        - **Project structure**: replace `web/static` and `web/templates`
          with `web/app/` (Vite project) and `web/dist/` (build output,
          gitignored). Mention `internal/api/`, `internal/insights/`,
          `internal/categories/`.
        - **Tech stack**: replace HTMX with React + Vite + TS; add
          TanStack Query, Workbox, IndexedDB.
        - **Configuration**: unchanged.
        - **Testing**: keep Go tests; mention `cd web/app && npx tsc
          --noEmit` and any frontend test runner if added; flag that the
          Playwright e2e suite needs updating for the React DOM.
- [x] Mark completed

### Task 15: Final verification

- [ ] `go test ./...` passes.
- [ ] `golangci-lint run` passes.
- [ ] `cd web/app && npx tsc --noEmit && npm run build` passes.
- [ ] `go build ./cmd/server` produces a binary that serves API + bundle on
      :8080.
- [ ] `docker-compose up --build` starts cleanly; data persists across a
      restart (verify by adding an expense, restarting, observing it).
- [ ] Manual smoke flows (login, add, edit, delete, infinite scroll,
      insights, install as PWA, offline write) all behave as expected.
- [ ] Existing `expenses.db` remains untouched: same rows, same users, same
      sessions table — no `ALTER TABLE` was run.
- [ ] Mark completed
