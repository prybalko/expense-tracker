# Expense Tracker

A simple, mobile-first expense tracking web application. Built with **Go**, **HTMX**, and **Pico CSS**.

## Features

- 📱 **Mobile-First Design**: Optimized for mobile usage with a responsive layout.
- ⚡ **Fast & Lightweight**: Server-side rendering with Go and HTMX for smooth interactions.
- 💰 **Expense Tracking**: Quick expense entry with a custom keypad.
- 📊 **Overview**: Daily grouping and monthly summaries.
- 🎨 **Modern UI**: Styled with [Pico CSS](https://picocss.com) v2.

## Project Structure

```
├── cmd/
│   └── server/           # Application entry point
├── internal/
│   ├── handlers/         # HTTP handlers and view logic
│   ├── models/           # Data models
│   └── storage/          # Database layer (SQLite)
├── web/
│   ├── static/           # Static assets (CSS)
│   └── templates/        # HTML templates
├── Dockerfile            # Multi-stage build
├── docker-compose.yml    # Docker composition
└── expenses.db           # SQLite database (ignored by git)
```

## Prerequisites

- **Go 1.25+** (for local development)
- **Docker** (optional, for containerized run)

## Quick Start

### Using Docker (Recommended)

```bash
docker-compose up --build
```
The app will be available at [http://localhost:8080](http://localhost:8080).

### Running Locally

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Run the application:
   ```bash
   go run ./cmd/server
   ```

3. Open your browser at [http://localhost:8080](http://localhost:8080).

## Tech Stack

- **Backend**: Go (Golang)
- **Database**: SQLite (embedded)
- **Frontend**: 
  - HTML Templates (Go `html/template`)
  - [HTMX](https://htmx.org) for interactivity
  - [Pico CSS](https://picocss.com) for styling

## License

MIT
