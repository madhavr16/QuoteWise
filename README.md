# QuoteWise

AI-assisted procurement and quotation intelligence for small and
medium-sized businesses, starting with IT hardware. QuoteWise turns
multiple vendor quotations into a structured, evidence-backed
comparison: AI extracts and explains, deterministic code calculates and
enforces rules, humans approve consequential decisions.

- Product requirements: [`PROJECT_SPEC.md`](PROJECT_SPEC.md)
- Architecture: [`docs/architecture.md`](docs/architecture.md)
- Implementation plan (milestone status): [`docs/implementation-plan.md`](docs/implementation-plan.md)
- Decision log: [`docs/decisions.md`](docs/decisions.md)

## Current status (Milestone 1 — local foundation)

Exists and works now: service skeletons for the Go API, the internal
Python AI service, and the React frontend; a Docker Compose development
stack (PostgreSQL+pgvector, Redis, MinIO, all three apps) with health
checks; validated configuration; structured JSON logging; a consistent
error envelope.

Deliberately not implemented yet (later milestones, per
`docs/implementation-plan.md`): authentication, procurement requests,
quotation upload/processing, extraction, comparison, RAG, reports. The
AI provider is mocked/absent by design — **no API keys or paid services
are required** for local development.

## Prerequisites

| Tool | Version | Needed for |
|---|---|---|
| Docker Desktop | engine 28.x, Compose v2 | The canonical dev stack — **start it before running the stack** (`docker ps` must succeed) |
| Go | ≥ 1.24 | Native `apps/api` development |
| Node.js + npm | 22.12+ (LTS) | Native `apps/web` development |
| Python | 3.12+ | Native `apps/ai-service` development (not installed on this machine — use Docker) |

## Quickstart (Docker Compose)

Run from the repository root:

```bash
cp .env.example .env
docker compose --env-file .env -f infra/compose.yaml up -d --build
```

Then check that everything is healthy:

```bash
docker compose --env-file .env -f infra/compose.yaml ps
```

| Service | URL / port |
|---|---|
| Web (dev server) | http://localhost:5173 |
| Go API liveness | http://localhost:8080/healthz |
| Go API readiness | http://localhost:8080/readyz |
| AI service liveness | http://localhost:8000/healthz |
| AI service readiness | http://localhost:8000/readyz |
| PostgreSQL | localhost:5432 (user/password/database in `.env`) |
| Redis | localhost:6379 |
| MinIO S3 API | localhost:9000 (no web console in recent community builds) |

Stop the stack (keeps dev data volumes):

```bash
docker compose --env-file .env -f infra/compose.yaml down
```

`down -v` also deletes the PostgreSQL/MinIO dev volumes — destructive,
only for a truly fresh start.

## Running apps natively (fallback)

Useful when iterating on one service. Start the infrastructure only:

```bash
docker compose --env-file .env -f infra/compose.yaml up -d postgres redis minio
```

Load `.env` into the shell (Git Bash on Windows):

```bash
set -a; . ./.env; set +a
```

Then run each app (each in its own terminal):

```bash
cd apps/api && go run ./cmd/api        # needs Go 1.24+
cd apps/web && npm install && npm run dev   # needs Node 22.12+
cd apps/ai-service && py -3.12 -m venv .venv \
  && . .venv/Scripts/activate \
  && pip install -r requirements-dev.txt \
  && uvicorn app.main:app --reload --port 8000   # needs Python 3.12+
```

The Vite dev server proxies `/api`, `/healthz`, and `/readyz` to
`http://localhost:8080` (override with `API_URL`).

## Development commands

| Area | Commands |
|---|---|
| Go API (`apps/api`) | `gofmt -l .` (no output = formatted) · `go vet ./...` · `go test ./...` · `go build ./...` |
| AI service (`apps/ai-service`) | `pytest` (Python 3.12+; or in Docker: `docker compose --env-file .env -f infra/compose.yaml run --rm ai-service pytest`) |
| Web (`apps/web`) | `npm run build` (type-checks with `tsc`, then builds) |
| Compose file | `docker compose --env-file .env -f infra/compose.yaml config` (validate without starting anything) |

## Repository layout

```text
QuoteWise/
├── apps/
│   ├── api/           # Go REST API (Gin; clean-architecture layers — ADR-015)
│   ├── ai-service/    # Python FastAPI internal service (3.12)
│   └── web/           # React + TypeScript + Vite frontend
├── infra/
│   └── compose.yaml   # local dev stack (pinned images, health checks)
├── docs/              # architecture, plan, decisions
├── .env.example       # placeholders only — copy to .env
└── PROJECT_SPEC.md
```

`testdata/` (synthetic quotations) and `scripts/` are created when
first needed in later milestones.

## Troubleshooting

- **`docker ps` fails / "cannot connect to the Docker daemon"** — start
  Docker Desktop and wait until `docker ps` succeeds. The Compose file
  can be validated without the daemon using the `config` command above.
- **Port conflicts (5432, 6379, 9000, 8000, 8080, 5173)** — override the
  `*_PORT` values in `.env`.
- **Python 3.12 missing** — run the AI service via Docker as shown
  above; it is the canonical environment anyway (ADR-013).
- **Slow hot-reload for the web dev server on Windows** — file watching
  across the Docker boundary is slower; run `npm run dev` natively
  instead if it bothers you.

## Security notes

- `.env` is git-ignored; `.env.example` contains placeholders only.
  Never commit real secrets.
- All credentials in `.env.example` are obvious development placeholders.
- The AI service is internal-only: the browser never talks to it
  directly; the Go API is the only intended caller.
- No AI provider credentials are configured — development runs fully
  offline with a mock provider by default (ADR-009).
