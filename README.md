<div align="center">

# Expense Tracker

Mobile-first expense tracker — Go API + React 19 PWA, designed to live on
your iOS home screen.

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=white)](https://react.dev)
[![Vite](https://img.shields.io/badge/Vite-7-646CFF?style=flat-square&logo=vite&logoColor=white)](https://vitejs.dev)
[![PWA](https://img.shields.io/badge/PWA-Installable-5A0FC8?style=flat-square&logo=pwa&logoColor=white)](https://web.dev/progressive-web-apps/)
[![SQLite](https://img.shields.io/badge/SQLite-Embedded-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

<table>
<tr>
<td><img src="screenshots/home.png" width="220" alt="Feed"/></td>
<td><img src="screenshots/insights.png" width="220" alt="Insights"/></td>
<td><img src="screenshots/new_expense.png" width="220" alt="Quick entry"/></td>
</tr>
</table>

</div>

## Highlights

- **Offline-first** — IndexedDB write queue + service worker; expenses
  added offline sync when you're back online.
- **Installable PWA** — standalone display, custom icons, safe-area
  aware.
- **Quick entry** — bespoke numpad keyed for fast logging.
- **Insights** — monthly totals and category breakdowns derived
  client-side from your feed.

## Quick start

### Docker

```bash
docker-compose up --build
```

Open <http://localhost:8080>.

### Local development

Two terminals — Go API on `:8080`, Vite dev server on `:5173` proxying
`/api`:

```bash
go run ./cmd/server
cd web/app && npm install && npm run dev
```

Open <http://localhost:5173>.

### Production build

```bash
cd web/app && npm run build          # outputs web/dist/, embedded by Go
go build -o expense-tracker ./cmd/server
./expense-tracker
```

## Configuration

| Variable         | Default       | Purpose                                |
| :--------------- | :------------ | :------------------------------------- |
| `PORT`           | `8080`        | HTTP port                              |
| `DB_PATH`        | `expenses.db` | SQLite file                            |
| `SECURE_COOKIE`  | `false`       | Set for HTTPS deployments              |
| `ADMIN_USER`     | `admin`       | Bootstrap admin (created on first run) |
| `ADMIN_PASSWORD` | *random*      | Printed to logs if unset               |

## Users

```bash
go run ./cmd/adduser -user <name> -password <pw> [-db expenses.db]
```

## Testing

```bash
go test ./...                        # backend + handlers
cd web/app && npm run build          # typecheck + frontend build

# E2E (Playwright via Go) — first time only:
go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.1
playwright install                   # add --with-deps on Linux
go test -v ./e2e/...
```

The E2E harness rebuilds `web/dist/` automatically when missing.
Selectors anchor on `data-testid`.

## Stack

Go 1.25 · SQLite via [`modernc.org/sqlite`](https://modernc.org/sqlite)
(CGo-free) · React 19 · Vite 7 · TanStack Query ·
[`vite-plugin-pwa`](https://vite-pwa-org.netlify.app/) + Workbox ·
IndexedDB writes via [`idb`](https://github.com/jakearchibald/idb) ·
Playwright.

See [`AGENTS.md`](AGENTS.md) for the full project layout and
contribution conventions.

## License

[MIT](LICENSE).
