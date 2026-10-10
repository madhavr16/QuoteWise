# QuoteWise Implementation Plan

- **Status:** Living document — updated at the end of every milestone
- **Requirements source of truth:** [`PROJECT_SPEC.md`](../PROJECT_SPEC.md)
- **Decision log:** [`decisions.md`](decisions.md)
- **Last updated:** 2026-10-10 (Milestone 0 complete)

---

## 1. Repository state snapshot (2026-10-10)

- **Contents:** `PROJECT_SPEC.md` only (929 lines). No application code,
  no README, no `.gitignore`, no `.env` files, no `docs/`, no CI
  configuration, no test fixtures.
- **Version control:** not a Git repository yet — no `.git` directory,
  no history, no branches, no uncommitted changes. There is no existing
  user code to preserve apart from `PROJECT_SPEC.md`.
- **Conventions:** none established; the spec's suggested monorepo
  layout (spec §14) is adopted as the target (ADR-002). No secrets are
  present anywhere in the repository.

## 2. Development tool inventory

| Tool | Requirement | Installed | Status |
|---|---|---|---|
| Git | any recent | 2.41.0 (Windows) | ✅ OK |
| Go | ≥ 1.24 | 1.24.2 (windows/amd64) | ✅ OK |
| Node.js / npm | 22 LTS | v22.13.1 / 10.5.0 | ✅ OK |
| Python | **≥ 3.12** (spec §5) | 3.11.4 — plus multiple installs (`C:\Python311`, `anaconda3`, WindowsApps shim) | ⚠️ **Gap** — use Docker (canonical) or install Python 3.12+ |
| Docker Engine | any recent | 28.0.1 | ⚠️ Installed, **daemon not running** (Docker Desktop not started) |
| Docker Compose | v2 | v2.33.1 | ✅ OK (once daemon starts) |
| pgvector / MinIO images | Milestone 1 | not yet pulled | ⏳ pulled by Compose in M1 |

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

### Milestone 1 — Local foundation ⬜

- [ ] Initialize Git repository, `.gitignore`, `.gitattributes`, initial
      commit (**requires explicit user authorization to commit**).
- [ ] Monorepo skeleton: `apps/web`, `apps/api`, `apps/ai-service`,
      `infra/`, `testdata/`, `scripts/`, root `README.md`.
- [ ] `.env.example` (placeholders only) + config validated at startup.
- [ ] `infra/compose.yaml`: PostgreSQL+pgvector, Redis, MinIO, app
      services; pinned image tags; health checks.
- [ ] Go API skeleton: router (ADR-003), `/healthz`, `/readyz`, JSON
      error envelope, structured logging.
- [ ] Python AI service skeleton: FastAPI, `/healthz`, Pydantic config,
      `MockProvider` stub (no network, no credentials).
- [ ] Frontend skeleton: Vite + React + TS minimal app, dev proxy.
- [ ] README quickstart; format/lint/test commands; CI command list.

**Acceptance:** a developer follows the README to start the project
locally; health checks work; secrets are not committed.

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

## 4. Milestone 1 ordered task queue (smallest first)

Each task ends with its verification step; do not proceed on a failed
command without diagnosing it.

1. **Git init** — `git init`, `.gitignore` (`.env`, `node_modules`,
   build outputs, Python venvs), `.gitattributes` (`* text=auto eol=lf`),
   initial commit of `PROJECT_SPEC.md` + `docs/`. *Blocked on user
   authorization to commit (spec §16 guardrails).*
2. **Skeleton directories + root README stub** — no scaffolding beyond
   what tasks 3–7 need.
3. **`.env.example`** — placeholder keys only (DB, Redis, MinIO, AI
   provider mode = `mock`).
4. **Compose stack** — postgres+pgvector, redis, minio (pinned tags);
   `docker compose ps` → all healthy.
5. **Go API skeleton** — `/healthz`, `/readyz` (checks DB), JSON error
   envelope, structured logs. Verify: `go vet`, `go build`, `go test`.
6. **AI service skeleton** — FastAPI `/healthz`, config, MockProvider
   stub. Verify: `pytest` (runs without credentials).
7. **Web skeleton** — Vite React TS app with API proxy. Verify:
   `npm run build`.
8. **README + commands** — documented start steps, format/lint/test
   commands for all three apps.

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
| R1 | Local Python is 3.11; spec requires 3.12+; three interpreters on PATH (system, Anaconda, WindowsApps shim) — wrong interpreter may be picked | ai-service won't run natively; env ambiguity | Canonical dev env is Docker (Python 3.12 image). If running natively, install 3.12+ and use `py -3.12`. `pyproject.toml` enforces `requires-python = ">=3.12"` |
| R2 | Docker daemon not running (Docker Desktop not started) | Compose stack unavailable | Start Docker Desktop before Milestone 1 work; Compose files can be authored regardless |
| R3 | No Git repository — work is unprotected | Lost changes, no history | `git init` + initial commit as the first M1 task (needs user authorization) |
| R4 | Windows dev quirks: CRLF/LF churn, path length limits, port conflicts (5432/6379/9000) | Noisy diffs, blocked ports | `.gitattributes` with LF from day one; document port overrides in `.env.example` |
| R5 | pgvector column dimension is fixed at DDL time by the embedding model | Painful migration if changed late | Choose embedding model (open decision P3) before the M5 migration; MockProvider uses a fixed deterministic dimension for tests |
| R6 | Real LLM/embedding provider unselected; cost and availability unknown | M3+ real-provider testing blocked; risk of accidental spend | MockProvider is the default everywhere (no credentials, no network); real provider is an explicit, user-approved decision (P2); no keys in repo |
| R7 | Scanned-PDF detection is heuristic; misclassification confuses users | Bad UX, silent data loss | Conservative detection; clear "OCR not supported yet" error; OCR adapter interface reserved (spec §2) |
| R8 | Prompt injection via quotation content | Wrong answers, leaked context | Document text treated as data; org filters before LLM; citation validation; injection tests in the eval set |
| R9 | Money precision bugs (floats, double tax, zero-fill) | Wrong comparisons — core product failure | NUMERIC columns + decimal library (ADR-005); rule-based engine with unit tests; unknown ≠ 0 |
| R10 | Extraction quality unknown without an evaluation set | Tuning blind, regressions unnoticed | Build the synthetic eval set early (M3/M5) and track metrics per spec §12 |
| R11 | Spec leaves DELETE semantics undefined | Unsafe data loss if implemented naively | No hard DELETE in MVP; status/archive-based; endpoint omitted until semantics are decided (ADR-012) |
| R12 | Secrets leakage (Windows clipboard history, `.env` in commits) | Security incident | `.env` git-ignored; `.env.example` placeholders only; startup validation with clear errors; never log keys or document contents |

## 7. Missing prerequisites checklist

- [ ] Git repository initialized (blocked on authorization to make the initial commit).
- [ ] Docker Desktop running (currently stopped).
- [ ] Python 3.12+ available (or accept Docker-only for the ai-service).
- [ ] `.gitignore`, `.gitattributes`, `.env.example`, `README.md` (Milestone 1).
- [ ] Pinned infrastructure image tags (Milestone 1, recorded in `decisions.md`).
- [ ] No LLM credentials configured — intentional; MockProvider covers dev/test until a provider is approved (P2).

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
- **T1** Go router confirmation (chi proposed) — before M1 scaffolding.
- **T2** Auth mechanism confirmation (server-side sessions proposed) — before M2.
- **T3** PostgreSQL major version and image pins — at M1.

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
