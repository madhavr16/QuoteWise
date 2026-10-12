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

## What currently exists (Milestone 2 part 1 — authentication)

Exists and works now: the full local foundation (service skeletons,
Compose stack with pinned images and health checks, validated config,
structured logging, error envelope) **plus authentication and
organization management** (Milestone 2 part 1, verified live
2026-10-12):

- Sign up (creates your organization + owner membership), sign in,
  sign out — argon2id passwords, server-side sessions in HttpOnly
  cookies (ADR-007)
- `/api/v1`: `POST /auth/register`, `POST /auth/login`, `POST
  /auth/logout`, `GET /me`, `GET /organizations/current`, `GET
  /organizations/current/members`
- Security: CSRF header on mutations, per-IP rate limiting on auth
  endpoints, login timing equalization, cross-organization scoping
- goose migrations applied automatically at API startup (ADR-016)

Deliberately not implemented yet (per `docs/implementation-plan.md`):
procurement requests, quotation upload/processing, extraction,
comparison, RAG, reports, member invites. The AI provider is
mocked/absent by design — **no API keys or paid services are
required** for local development.

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

Then check that everything is running (app services should show
`(healthy)`; MinIO shows no health status because its image is
distroless — use its live endpoint below):

```bash
docker compose --env-file .env -f infra/compose.yaml ps
```

| Service | URL / port |
|---|---|
| Web (dev server) | http://localhost:5173 |
| Go API liveness | http://localhost:18069/healthz |
| Go API readiness | http://localhost:18069/readyz |
| AI service liveness | http://localhost:8000/healthz |
| AI service readiness | http://localhost:8000/readyz |
| PostgreSQL | localhost:5432 (user/password/database in `.env`) |
| Redis | localhost:6379 |
| MinIO S3 API | localhost:9000 — liveness: http://localhost:9000/minio/health/live (distroless image; no in-container healthcheck) |

Stop the stack (keeps dev data volumes):

```bash
docker compose --env-file .env -f infra/compose.yaml down
```

`down -v` also deletes the PostgreSQL/MinIO dev volumes — destructive,
only for a truly fresh start.

### Try the auth flow (optional)

```bash
BASE=http://localhost:18069
JAR=$LOCALAPPDATA/Temp/opencode/qw-cookies.txt

# Register (creates account + organization + session cookie)
curl -i -c "$JAR" -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" -H "X-Requested-With: XMLHttpRequest" \
  -d '{"email":"you@example.com","password":"a-12+-char-password","organization_name":"Your Org"}'

# Signed-in requests (cookie jar)
curl -b "$JAR" "$BASE/api/v1/me"
curl -b "$JAR" "$BASE/api/v1/organizations/current/members"

# Sign out
curl -b "$JAR" -X POST -H "X-Requested-With: XMLHttpRequest" "$BASE/api/v1/auth/logout"

# Or just use the web app at http://localhost:5173 — it has login
and registration forms.
```

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
`http://localhost:8080` by default. If you changed `HTTP_PORT` (this
machine uses 18069, since 8080 is occupied), start it with
`API_URL=http://localhost:18069 npm run dev`.

## Development commands

| Area | Commands |
|---|---|
| Go API (`apps/api`) | `gofmt -l .` (no output = formatted) · `go vet ./...` · `go test ./...` · `go build ./...` |
| Go store integration (`apps/api`) | `TEST_DATABASE_URL=postgresql://quotewise:quotewise-dev@localhost:15432/quotewise_test go test ./internal/store/postgres/` (skips when the variable is unset) |
| AI service (`apps/ai-service`) | `pytest` (Python 3.12+; or in Docker: `docker compose --env-file .env -f infra/compose.yaml run --rm --entrypoint pytest ai-service -p no:cacheprovider` — `--entrypoint` overrides the uvicorn ENTRYPOINT) |
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
- **Port conflicts (5432, 6379, 9000, 8000, 18069, 5173)** — override the
  `*_PORT` values in `.env`.
- **Python 3.12 missing** — run the AI service via Docker as shown
  above; it is the canonical environment anyway (ADR-013).
- **Slow hot-reload for the web dev server on Windows** — file watching
  across the Docker boundary is slower; run `npm run dev` natively
  instead if it bothers you.
- **Docker Desktop stops answering (`docker ps` hangs / engine-pipe 500)**
  — this machine has hit that periodically. Restart it:
  `taskkill //F //IM "Docker Desktop.exe"; taskkill //F //IM "com.docker.backend.exe"`, relaunch Docker Desktop, wait ~40s. Containers and volumes survive the restart.
- **Go build/test fails with out-of-memory while the full Docker stack runs** — stop the stack first (`docker compose ... stop`), build, then `start` it again. The first heavy build warms the cache; Docker image rebuilds are layer-cached.
- **A native Windows PostgreSQL owns port 5432 on the host** — that is why `POSTGRES_PORT` is 15432 (same pattern as `HTTP_PORT=18069`). Host-side connections (integration tests, native runs) must use 15432; traffic inside the Compose network is unaffected.

## Security notes

- `.env` is git-ignored; `.env.example` contains placeholders only.
  Never commit real secrets.
- All credentials in `.env.example` are obvious development placeholders.
- The AI service is internal-only: the browser never talks to it
  directly; the Go API is the only intended caller.
- No AI provider credentials are configured — development runs fully
  offline with a mock provider by default (ADR-009).
