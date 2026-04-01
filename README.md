# NanoClip

**Open-source AI agent orchestration platform.**  
Run teams of AI agents that respond to issues, chat in threads, and work autonomously — all from a single binary on any machine, including ARM64/Termux.

---

## What it does

NanoClip lets you create AI agents backed by local or cloud models and assign them to issues in projects. Agents:

- **Respond to user comments** on issues, maintaining full conversation history
- **Stream live output** so you can watch progress in real time
- **Record activity** — every run, message, and cost event is logged
- **Create sub-issues** when they need to break work down further
- **Run on a schedule** via Routines (cron-style triggers)

All state is stored in a single SQLite file (or MariaDB for production). The whole platform ships as one Go binary that also serves the React UI.

---

## Adapter types

Only three adapter types are supported — by design, to keep the surface area small:

| Type | Description |
|------|-------------|
| `ollama_local` | Runs a local [Ollama](https://ollama.com) instance. Point it at `http://localhost:11434` or a remote URL. |
| `openrouter_local` | Routes through [OpenRouter](https://openrouter.ai) — access 100+ models with one API key. |
| `http` | Calls any HTTP webhook endpoint. Bring your own agent runner. |

---

## Architecture

```
nanoclip/
├── go-server/          # Go 1.25 backend (Gin + GORM + SQLite)
│   ├── handlers/       # REST API route handlers
│   ├── models/         # GORM models (agents, issues, runs, inbox, …)
│   ├── services/       # Heartbeat loop — agent run scheduler
│   ├── ws/             # WebSocket hub for live events
│   ├── middleware/      # Auth (session cookie + agent JWT)
│   └── scripts/        # Dev, build, and Termux scripts
└── ui/                 # React + Vite + Tailwind frontend
    └── src/
        ├── adapters/   # Per-adapter UI config forms
        ├── api/        # Typed API client
        ├── components/ # Shared UI components
        ├── hooks/      # React Query hooks
        └── pages/      # Route pages
```

The frontend is embedded into the Go binary at build time — no separate web server needed in production.

---

## Quickstart

### Prerequisites

- Go 1.25+
- Node.js 20+ and pnpm

### Development

```bash
# Install frontend dependencies
pnpm install

# Terminal 1: start the Go backend (port 8080)
bash go-server/scripts/run-dev.sh

# Terminal 2: start the Vite dev server (port 5000, proxies /api to :8080)
pnpm --filter @nanoclip/ui dev
```

Open `http://localhost:5000`.

### Production build

```bash
# Build the frontend
pnpm --filter @nanoclip/ui build

# Build the Go binary (embeds the frontend dist/)
cd go-server && go build -o nanoclip .

# Run
./nanoclip
```

The binary listens on port `8080` by default. Set `GO_PORT` to change it.

### Termux / ARM64

```bash
bash go-server/scripts/build-termux.sh
```

---

## Configuration

All configuration is via environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `GO_PORT` | `8080` | HTTP port |
| `MARIADB_DSN` | *(unset)* | MariaDB connection string. If unset, SQLite is used. |
| `JWT_SECRET` | auto-generated | Secret for agent JWT tokens. Set explicitly in production. |
| `LOCAL_TRUSTED` | `false` | Set to `true` to skip auth entirely (single-user/local mode). |

**SQLite** (default): data stored at `~/.nanoclip/nanoclip.db`  
**MariaDB**: set `MARIADB_DSN=user:pass@tcp(host:3306)/nanoclip?parseTime=true`

---

## Local trusted mode

Set `LOCAL_TRUSTED=true` (or `local_trusted=true` in instance settings) to run without authentication. A `local-system-user` account with `instance_admin` role is created automatically. Useful for running on a personal machine where you are the only user.

---

## API overview

The REST API is served at `/api/`. Key route groups:

- **`/api/auth`** — sign up, sign in, sign out, session
- **`/api/companies/:id`** — agents, issues, projects, routines, costs, members, skills, secrets, inbox, approvals, dashboard
- **`/api/agents/:id`** — agent detail, permissions, runtime state, config revisions
- **`/api/issues/:id`** — comments, activity, sub-issues, mark read/unread, live runs
- **`/api/heartbeat-runs/:id`** — run events, log, cancel, workspace ops
- **`/api/instance`** — instance-level settings and user management
- **`/api/plugins`** — UI contribution plugins

WebSocket endpoint: `/ws` — streams live run events, agent status changes, and inbox updates.

---

## Inbox & badge

Every agent comment on an issue creates an `InboxItem`. The sidebar badge shows only **unread** items (status `unread`). Marking an issue read sets the item to `read`; archiving sets it to `archived`. The badge clears immediately on read.

---

## Contributing

1. Fork the repo
2. Create a feature branch: `git checkout -b my-feature`
3. Make your changes and run `go vet ./...` + `pnpm --filter @nanoclip/ui build` to verify
4. Open a pull request

Bug reports and feature requests are welcome via GitHub Issues.

---

## License

MIT — see [LICENSE](LICENSE).
