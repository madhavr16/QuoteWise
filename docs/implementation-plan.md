# QuoteWise Implementation Plan

- **Status:** Living document — updated at the end of every milestone
- **Requirements source of truth:** [`PROJECT_SPEC.md`](../PROJECT_SPEC.md)
- **Decision log:** [`decisions.md`](decisions.md)
- **Last updated:** 2026-10-11 (Milestone 1 in progress)

---

## 1. Repository state snapshot (2026-10-10, Milestone 1)

- **Contents:** Milestone 0 docs plus the Milestone 1 skeleton —
  `apps/api` (Go/Gin), `apps/ai-service` (Python/FastAPI),
  `apps/web` (React/Vite), `infra/compose.yaml`, `.gitignore`,
  `.gitattributes`, `.env.example`, `README.md`.
- **Version control:** Git repository on `main` with remote `origin`
  (github.com/madhavr16/QuoteWise); Milestone 0 work is committed.
  Milestone 1 changes are uncommitted, pending explicit approval.
- **Conventions:** monorepo layout per ADR-002; LF line endings via
  `.gitattributes`; no secrets committed (`.env` is git-ignored and
  confirmed excluded via `git check-ignore`; `.env.example` holds
  placeholders only).

## 2. Development tool inventory

| Tool | Requirement | Installed | Status |
|---|---|---|---|
| Git | any recent | 2.41.0 (Windows) | ✅ OK |
| Go | ≥ 1.24 | 1.24.2 (windows/amd64) | ✅ OK |
| Node.js / npm | 22 LTS | v22.13.1 / 10.5.0 | ✅ OK |
| Python | **≥ 3.12** (spec §5) | 3.11.4 (3.12 not installed) | ⚠️ Gap — ai-service smoke-tested on 3.11; canonical 3.12 runs via Docker |
| Docker Engine | any recent | 28.0.1 | ❌ **Blocker: daemon not running** (Docker Desktop not started) — image pulls, `up`, and container health checks unverified |
| Docker Compose | v2 | v2.33.1 | ✅ OK — `docker compose config` validation works without the daemon |
| pgvector / MinIO images | Milestone 1 | not yet pulled | ⏳ **Blocked: Docker daemon down** — tags verified online (ADR-011); pulls pending |

## 3. Milestones

Status legend: ✅ complete · 🔶 in progress · ⬜ not started.

### Milestone 0 — Repository reconnaissance and plan ✅

- [x] Inspect repository, Git status, and installed tools (results in §1–§2).
- [x] Create `docs/architecture.md`.
- [x] Create `docs/implementation-plan.md`.
- [x] Create `docs/decisions.md`.
- [x] Record risks, missing prerequisites, and unresolved decisions (§6–§8).

**Acceptance (spec §13):** a documented architecture and ordered
implementation plan exist; no unnecessary code was generated. ✅

### Milestone 1 — Local foundation 🔶 (code complete; stack run pending Docker)

- [x] Git repository already initialized by the owner (branch `main`,
      remote `origin`); `.gitignore` + `.gitattributes` created (LF
      everywhere; `.env` confirmed ignored via `git check-ignore`).
      Committing the M1 work **requires explicit user authorization**.
- [x] Monorepo skeleton: `apps/web`, `apps/api`, `apps/ai-service`,
      `infra/`, root `README.md` (`testdata/`, `scripts/` are created
      when first needed in later milestones).
- [x] `.env.example` (placeholders only) + config validated at startup
      (Go `config.Load`, Python `Settings`).
- [x] `infra/compose.yaml`: PostgreSQL+pgvector, Redis, MinIO
      (quay.io — see ADR-011), app services; pinned image tags; health
      checks. Validated with `docker compose config`; **not yet run**.
- [x] Go API skeleton: Gin (ADR-003), clean-architecture layers
      (ADR-015: server/handler/service/domain/store/gateway +
      composition root), `/healthz`, `/readyz` (postgres + ai_service
      probes), JSON error envelope, structured logging, graceful
      shutdown. Verified: gofmt clean, `go vet`, `go build`,
      `go test ./...` pass.
- [x] Python AI service skeleton: FastAPI with mirrored layers
      (ADR-015: server/api/routes/services/providers), `/healthz`,
      `/readyz`, Pydantic config, JSON logging, `/internal/v1` 501
      stubs. Verified: `pytest` 9/9 (**on Python 3.11 — non-canonical
      smoke test**; canonical 3.12 run pending Docker), no credentials
      needed.
- [x] Frontend skeleton: Vite + React + TS status page with explicit
      loading/error states, dev proxy to the Go API. Verified:
      `npm install` + `npm run build` (tsc + vite) pass.
- [x] README quickstart; format/lint/test commands documented.

**Acceptance:** a developer follows the README to start the project
locally; health checks work; secrets are not committed.
⏳ **Not yet verified** — requires the Docker engine, which is not
running on this machine. Remaining steps in §4a.

### Milestone 2 — Persistence and procurement workflow ⬜

Database migrations; organizations/users/members + authorization;
procurement request CRUD; quotation metadata + private file storage;
upload validation and processing-job state; dashboard and request
detail screens.
**Acceptance:** a signed-in user can create a request and securely
upload a quotation.

### Milestone 3 — Document extraction and human review ⬜

PDF/DOCX/XLSX/CSV parsers; extraction interface with mock + real
provider adapters; persist structured extraction and evidence; review/
edit/confirm UI; failure handling, retries, idempotency.
**Acceptance:** a user can upload a supported quotation, see extracted
values, correct them, and confirm them.

### Milestone 4 — Deterministic comparison engine ⬜

Decimal-safe money calculations; comparable-total rules and warnings;
IT hardware spec normalization; side-by-side matrix; unit and
integration tests.
**Acceptance:** identical input data produces identical comparison
results; unknown or incompatible values are visibly flagged.

### Milestone 5 — RAG and evidence-backed Q&A ⬜

Chunking with source locations; embeddings + pgvector migration; scoped
retrieval and grounded answer generation; citations and uncertainty
responses; retrieval/citation evaluation tests with provider mocks.
**Acceptance:** answers cite source material, missing evidence is
acknowledged, cross-organization retrieval tests pass.

### Milestone 6 — Reports and product polish ⬜

CSV export and print-friendly report; loading/empty/failure states;
audit trail for important edits; security review; demo data and product
walkthrough.
**Acceptance:** the core user journey works end to end on a fresh local
setup.

### Milestone 7 — Deployment ⬜

Deployment target selection; dev/prod config separation; private object
storage; managed/persistent PostgreSQL; HTTPS, secrets, migrations,
backups, logs, health checks, error monitoring; CI/CD + rollback;
production-like smoke test.
**Acceptance:** accessible over HTTPS, secrets protected, data persists
across restarts, setup/rollback documented.

## 4. Milestone 1 ordered task queue — statuses (2026-10-10)

Each task ends with its verification step; do not proceed on a failed
command without diagnosing it.

1. ~~Git init~~ — repository already initialized by the owner; M1 files
   are uncommitted (commit pending explicit authorization).
2. [x] Skeleton directories + root README (2026-10-10).
3. [x] `.env.example` — placeholders only; no AI provider keys by
   design (mock provider, ADR-009).
4. [x] Compose stack authored and validated (`docker compose config`);
   **run pending the Docker engine** (see §4a).
5. [x] Go API skeleton — verified: gofmt/vet/build/test all pass.
6. [x] AI service skeleton — verified: pytest 8/8 (Python 3.11 smoke
   test; canonical 3.12 via Docker pending).
7. [x] Web skeleton — verified: `npm install` + `npm run build` pass.
8. [x] README + commands documented.

### 4a. Remaining steps to close Milestone 1 (blocked on Docker)

1. Start Docker Desktop; confirm `docker ps` succeeds.
2. `docker compose --env-file .env -f infra/compose.yaml up -d --build`
   → `docker compose ps`: all services healthy.
3. Verify endpoints: `curl http://localhost:8080/healthz`,
   `curl http://localhost:8080/readyz` (postgres + ai_service checks
   ok), `curl http://localhost:8000/healthz`,
   `curl http://localhost:8000/readyz`, and `http://localhost:5173`
   loads.
4. Confirm the pinned MinIO quay.io tag pulls; if the tag is missing,
   pin the next release tag from MinIO's GitHub releases and note it.
5. Canonical test run in Python 3.12:
   `docker compose --env-file .env -f infra/compose.yaml run --rm ai-service pytest`.
6. Mark the acceptance criteria in §3 as verified; commit the
   Milestone 1 work (with explicit approval).

## 5. Testing strategy (summary, per spec §12)

- **Go:** request validation, org access control, review rules, price
  normalization and totals, missing tax/shipping handling, currency
  mismatch behavior, comparison warnings, API error responses.
- **Python:** extraction schema validation, parser fixtures for all four
  formats, empty/scanned/corrupted behavior, unknown fields stay
  unknown, evidence preservation, retrieval restricted to authorized
  sets, citations resolve to retrieved chunks, prompt-injection strings
  do not change behavior, mock-provider tests need no credentials.
- **Integration:** upload → job → extraction → review → comparison;
  idempotent reprocessing; retryable failures; cross-organization
  isolation; export matches comparison API; Compose health checks.
- **AI evaluation:** small version-controlled synthetic set (different
  layouts, missing/contradictory totals, tax-inclusive and exclusive,
  discounts/shipping, spec variants, warranty/delivery terms, unanswerable
  questions, malicious embedded instructions). Track field accuracy,
  citation correctness, retrieval relevance, groundedness, latency,
  approximate cost. Build it before tuning retrieval parameters.

## 6. Technical risks and mitigations

| # | Risk | Impact | Mitigation |
|---|---|---|---|
| R1 | Local Python is 3.11; spec requires 3.12+; three interpreters on PATH (system, Anaconda, WindowsApps shim) — wrong interpreter may be picked | ai-service won't run natively; env ambiguity | Mitigated for M1: pytest 8/8 smoke-tested on 3.11; canonical 3.12 runs in Docker (`python:3.12.15-slim-bookworm`, exact pin). Native runs still need 3.12+ |
| R2 | **Active blocker:** Docker daemon not running (Docker Desktop not started) | Image pulls, `compose up`, live health checks, and the canonical Python 3.12 test run are unverified | Everything engine-independent is done (`docker compose config` validates the stack). Start Docker Desktop and run §4a steps 1–5 |
| R3 | Milestone 1 work is uncommitted | Lost changes if interrupted | Repository is initialized (`main`, remote `origin`); commit M1 files with explicit user approval |
| R4 | Windows dev quirks: CRLF/LF churn, path length limits, port conflicts (5432/6379/9000/5173) | Noisy diffs, blocked ports | Mitigated: `.gitattributes` forces LF; every port is overridable via `.env` |
| R5 | pgvector column dimension is fixed at DDL time by the embedding model | Painful migration if changed late | Choose embedding model (open decision P3) before the M5 migration; MockProvider uses a fixed deterministic dimension for tests |
| R6 | Real LLM/embedding provider unselected; cost and availability unknown | M3+ real-provider testing blocked; risk of accidental spend | MockProvider is the default everywhere (no credentials, no network); real provider is an explicit, user-approved decision (P2); no keys in repo |
| R7 | Scanned-PDF detection is heuristic; misclassification confuses users | Bad UX, silent data loss | Conservative detection; clear "OCR not supported yet" error; OCR adapter interface reserved (spec §2) |
| R8 | Prompt injection via quotation content | Wrong answers, leaked context | Document text treated as data; org filters before LLM; citation validation; injection tests in the eval set |
| R9 | Money precision bugs (floats, double tax, zero-fill) | Wrong comparisons — core product failure | NUMERIC columns + decimal library (ADR-005); rule-based engine with unit tests; unknown ≠ 0 |
| R10 | Extraction quality unknown without an evaluation set | Tuning blind, regressions unnoticed | Build the synthetic eval set early (M3/M5) and track metrics per spec §12 |
| R11 | Spec leaves DELETE semantics undefined | Unsafe data loss if implemented naively | No hard DELETE in MVP; status/archive-based; endpoint omitted until semantics are decided (ADR-012) |
| R12 | Secrets leakage (Windows clipboard history, `.env` in commits) | Security incident | `.env` git-ignored; `.env.example` placeholders only; startup validation with clear errors; never log keys or document contents |
| R13 | gin v1.12 and pgx v5.9+ require Go ≥ 1.25 while the installed toolchain is 1.24.2 | Dependency drift between local toolchain and latest libraries | Pinned gin v1.11.0 + pgx v5.8.0 (newest Go-1.24-compatible); toolchain upgrade is a deliberate, user-approved change (ADR-011) |

## 7. Missing prerequisites checklist

- [x] Git repository initialized (M0 committed; M1 commit pending approval).
- [ ] Docker Desktop running — **blocker**; engine currently down.
- [ ] Python 3.12+ available (accept Docker-only for the ai-service for now).
- [x] `.gitignore`, `.gitattributes`, `.env.example`, `README.md` created.
- [x] Pinned infrastructure image tags recorded in `decisions.md` (ADR-011); pulls pending Docker.
- [x] No LLM credentials configured — intentional; MockProvider covers dev/test until a provider is approved (P2).

## 8. Unresolved product and technical decisions

Tracked with defaults and needed-by dates in
[`decisions.md`](decisions.md) §Open decisions:

- **P1** Initial customer persona (SMB IT-hardware buyer vs IT reseller vs
  procurement consultant) — shapes UX and data model emphasis.
- **P2** Real LLM/embedding provider and budget (paid; explicit approval
  required).
- **P3** Embedding model and vector dimension (blocks the M5 migration).
- **P4** Deployment target for Milestone 7.
- **P5** Data deletion/retention semantics, including objects and embeddings.
- **P6** Organization signup mechanics (first-user-creates-org proposed).
- **T2** Auth mechanism confirmation (server-side sessions proposed) — before M2.

Resolved in Milestone 1: **T1** (router: Gin, ADR-003) and **T3** (version pins, ADR-011).

## 9. Definition of done for the MVP (spec §17)

A user can create an IT-hardware procurement request, upload quotations
in supported formats, review and correct extractions, compare side by
side with deterministic and transparent calculations, receive grounded
AI answers with verifiable citations, and export a report — with
organization isolation tested, automated tests in place, documented
local setup, a controlled deployment with protected secrets and
persistent data, and an honest README.

## 10. Change log

- **2026-10-10 — Milestone 0 complete.** Repository inspected; architecture,
  implementation plan, and decision log created. No code, dependencies,
  or configuration files added.
- **2026-10-10 — Milestone 1 implemented (stack run pending).**
  Skeletons for `apps/api` (Gin), `apps/ai-service` (FastAPI), and
  `apps/web` (React/Vite); `infra/compose.yaml` with pinned images;
  `.gitignore`, `.gitattributes`, `.env.example`, `README.md`. Router
  confirmed as Gin (ADR-003); all version pins recorded (ADR-011).
  Verified: Go fmt/vet/build/test pass; pytest 8/8 (3.11 smoke);
  `npm install` + `npm run build` pass; `docker compose config`
  validates the stack. Unverified (Docker engine down): image pulls,
  `compose up`, live health checks, canonical 3.12 test run — see §4a.
- **2026-10-11 — Clean-architecture restructure (ADR-015).** Both
  backends reorganized into explicit layer folders: Go (`cmd/api`
  composition root; `internal/server`, `handler`, `service`, `domain`,
  `store/postgres`, `gateway/aiservice`, `version`, `config`) and
  Python (`server/`, `api/routes/`, `services/`, `providers/`).
  Behavior unchanged (health endpoints, config validation, logging);
  all checks re-run and pass (Go fmt/vet/build/test; pytest 9/9 on
  Python 3.11).
