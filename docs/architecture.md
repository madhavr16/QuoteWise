# QuoteWise Architecture

- **Status:** Proposed (Milestone 0 deliverable)
- **Requirements source of truth:** [`PROJECT_SPEC.md`](../PROJECT_SPEC.md)
- **Last updated:** 2026-10-12 (Milestone 2 part 1: auth + organization management implemented)

QuoteWise turns multiple vendor quotations (PDF, DOCX, XLSX, CSV) into a
structured, evidence-backed comparison with AI-assisted extraction and
RAG question answering. Deterministic application code performs all
calculations and rule enforcement; humans review and confirm every
consequential value.

> **Product principle (spec §1):** AI extracts, retrieves, and explains.
> Deterministic application code validates, calculates, and enforces rules.
> Humans approve consequential decisions.

---

## 1. System overview

```
                React + TypeScript Web App  (apps/web)
                               |
                         HTTPS / REST  (/api/v1)
                               |
                  Go API (apps/api)   (Gin; ADR-003)
                /          |            \
        PostgreSQL     Redis          Object Storage
        + pgvector   cache/rate-    (MinIO in dev,
        (system of    limiting       S3-compatible in prod)
         record)      (optional)     private buckets
              \          |            /
               \    Background jobs (PostgreSQL-backed; ADR-004)
                \         |
                 Python AI Service (apps/ai-service)
             document parsing / extraction / embeddings / RAG
                               |
                   LLM + embedding provider
              (deterministic MockProvider in dev/tests; ADR-009)
```

### Boundaries

1. The browser talks **only** to the Go API over HTTPS under `/api/v1`.
   It never calls the Python AI service directly (spec §9).
2. The Go API is the business authority: authentication, organization
   scoping, validation, CRUD, comparison math, exports, and audit.
3. The Python AI service is an internal, non-public service owning
   document parsing and AI workflows. Its endpoints live under
   `/internal/v1` and require service authentication.
4. PostgreSQL is the only source of truth. Redis is optional
   infrastructure (cache, rate limiting) and never a required dependency
   for core flows. Object storage holds original files; the database
   stores generated storage keys, never file bytes.

---

## 2. Components

### 2.1 Frontend — `apps/web` (Milestone 1+)

- React + TypeScript + Vite; Tailwind CSS; React Router;
  TanStack Query for server state; React Hook Form + Zod for forms.
- Client-side validation with Zod is a UX nicety; the Go API re-validates
  everything server-side.
- Pages (spec §10): Sign in/Register, Dashboard, Request list,
  Request create/edit, Request detail, Quotation review, Comparison
  workspace, Ask QuoteWise (Q&A), Report view.
- Every page must implement explicit loading, empty, and error states.

### 2.2 Go API — `apps/api` (Milestone 1+)

- Go 1.24. Router: **Gin** (user-confirmed 2026-10-10; ADR-003).
- Clean architecture layering (ADR-015). Dependencies point inward
  only: `handler`/`server` (transport) → `service` (use cases) →
  `domain`; adapters in `store`/`gateway` implement the ports the
  services define. `cmd/api` is the composition root — the only place
  that knows every layer.

```
apps/api/
├── cmd/api/                    # composition root: config, wiring, lifecycle
├── migrations/                 # embedded goose SQL migrations (up-only; ADR-016)
└── internal/
    ├── config/                 # env parsing, validated at startup
    ├── domain/                 # entities + business rules (user/org/session now; comparison engine M4)
    ├── service/                # use cases + ports (health, auth, organizations; requests/quotations later)
    ├── handler/                # thin HTTP handlers + response DTOs
    ├── server/                 # Gin engine, middleware, error envelope, routes
    ├── store/                  # persistence adapters (postgres: users, orgs, memberships, sessions; M2 part 2)
    ├── gateway/                # external service adapters (aiservice now)
    ├── security/               # auth-crypto adapters (argon2id hasher, token generator)
    ├── version/                # build metadata
    ├── storage/                # S3-compatible object storage abstraction (M2+)
    ├── jobs/                   # worker: claim, run, retry processing_jobs rows (M3+)
    └── audit/                  # audit event recording (M6+)
```

- Handlers stay thin. No LLM orchestration or document parsing inside
  HTTP handlers. No business rules in handlers or in the server
  layer; services orchestrate, adapters do I/O.
- Milestone 2 part 1 endpoints (all under `/api/v1`):
  `POST /auth/register` (creates org + owner, 409 on duplicate email),
  `POST /auth/login` (200 + session cookie), `POST /auth/logout`
  (204), `GET /me` (200 with user/org/role, 401 if unauthenticated),
  `GET /organizations/current` (200), `GET
  /organizations/current/members` (200). Mutating requests require
  `X-Requested-With: XMLHttpRequest`; auth endpoints are rate-limited
  per IP (ADR-017).
- Money: decimal arithmetic only (`shopspring/decimal`), serialized as
  strings in JSON APIs (ADR-005).
- Background worker runs in-process and claims jobs from the
  `processing_jobs` table (ADR-004).

### 2.3 Python AI service — `apps/ai-service` (Milestone 1+)

- Python **3.12+** (target) + FastAPI + Pydantic v2.
- Layered like the Go API (ADR-015): `server/` (app factory, middleware,
  error handlers), `api/routes/` (thin handlers), `services/` (use
  cases), `providers/` (LLM/embedding adapters incl. the deterministic
  `MockProvider`, M3), plus `parsers/`, `schemas/`, `retrieval/`, and
  `answering/` as they land in Milestones 3 and 5. `main.py` is the
  composition root.
- Heavy parsing runs off the event loop (thread/process pool) so request
  handling is never blocked.
- Never exposed to the public internet; internal endpoints require
  service authentication; browser cannot reach it.
- Pydantic validates every extraction response. Missing values are
  `null`/unknown, never fabricated defaults.

### 2.4 Data stores

| Store | Role | Notes |
|---|---|---|
| PostgreSQL 16 + pgvector | System of record | Migrations from Milestone 2; vector column added Milestone 5 (dimension fixed by chosen embedding model, ADR-010) |
| Redis 7 | Optional cache, rate limiting | Never source of truth; app must run without it |
| Object storage | Original quotation files | MinIO (dev) / S3-compatible (prod); private buckets, generated keys (`org/quote/uuid`), sha256 checksums recorded in DB |

---

## 3. Key runtime flows

### 3.1 Quotation upload → extraction (async, idempotent)

Quotation status machine: `uploaded → processing → needs_review → ready`
with `failed` as a terminal-but-retryable state.

1. Browser → `POST /api/v1/procurement-requests/{id}/quotations`.
2. Go API authenticates the user, verifies organization membership and
   request ownership, validates file size and content type by sniffing
   bytes (never trusting client MIME alone), computes the sha256
   checksum, generates a storage key, uploads the object to private
   storage, inserts the `quotations` row (`uploaded`) and a
   `processing_jobs` row with an idempotency key, and returns promptly
   (HTTP 202) with the quotation and job IDs.
3. The worker claims the pending job (row claim with `FOR UPDATE SKIP
   LOCKED` semantics), marks it `processing`, and calls the Python
   service `POST /internal/v1/documents/extract` with a timeout.
4. Python parses the document via a format adapter, detects empty or
   likely-scanned documents, sends relevant text to the configured LLM
   (or MockProvider) for schema-constrained extraction, validates the
   response with Pydantic, and returns structured fields + evidence +
   confidence. It never guesses missing values.
5. Go persists structured results and evidence, sets the quotation to
   `needs_review`, marks the job `completed`, and writes an audit event.
6. On failure the job records a sanitized error; the user can retry via
   `POST /api/v1/quotations/{id}/retry-processing`. Retries never
   overwrite user-confirmed values without explicit user action
   (spec §4.3). Reprocessing is idempotent.

Long-running document processing never blocks an HTTP request.

### 3.2 Comparison (deterministic, no LLM)

- A pure Go domain service takes reviewed quotations and produces a
  comparison matrix of values **plus assumptions and warnings**.
- Rules (spec §7): preserve vendor-quoted totals separately from
  recalculated ones; distinguish subtotal/discount/tax/shipping; never
  assume tax-inclusive vs exclusive; never treat missing charges as
  zero; never add tax twice; validate arithmetic and flag discrepancies
  against the vendor's stated total; mark normalized totals incomplete
  and explain why when data is missing; compare like-for-like products,
  quantities, units, and currencies only; flag substitutions and spec
  mismatches; preserve original source values.
- One currency per procurement request in the MVP; mismatched
  quotations are flagged as not directly comparable, never silently
  converted.
- The LLM never performs final arithmetic.

### 3.3 RAG question answering (spec §8)

1. User asks a question scoped to one procurement request.
2. Go API authenticates and authorizes; identifies the request and its
   allowed documents (organization filter applied before anything
   reaches the LLM).
3. Retrieval combines structured DB fields and document chunks;
   initially PostgreSQL full-text search, with pgvector similarity added
   in Milestone 5; hybrid retrieval only if evaluation results improve.
4. Only relevant context and instructions go to the LLM, with token and
   time limits.
5. Answers must cite retrieved evidence; cited source IDs are validated
   to belong to the retrieved set; missing or contradictory evidence
   produces an explicit uncertainty response.
6. Uploaded document content is treated as **data, not instructions** —
   prompt-injection strings inside quotations must not alter system
   behavior.

---

## 4. Security architecture

- **Tenancy:** every organization-owned row carries `organization_id`;
  every query filters on it; membership is verified server-side on every
  request; an organization ID from the client is never trusted.
- **Authentication:** server-side opaque sessions (ADR-007, verified live
  2026-10-12): argon2id hashing (64 MiB, t=2, p=2, PHC-encoded), 256-bit
  tokens stored only as SHA-256 hashes, HttpOnly SameSite=Lax cookie with
  sliding 168h TTL, logout revocation, timing-equalized login failures;
  no custom crypto.
- **Upload hardening:** size cap, extension allowlist plus magic-byte
  sniffing, generated storage keys (never user paths), checksums,
  path-traversal-safe object access, never executing uploaded content,
  scanned PDFs rejected with a clear "OCR not supported yet" message
  (adapter interface reserved for future OCR).
- **File privacy:** private buckets; no public object URLs; downloads
  via authenticated endpoints.
- **AI safety:** organization/request filters before retrieval results
  reach the LLM; citation resolution enforced; timeouts and token limits;
  document contents, API keys, and tokens never logged by default;
  no training on customer quotations; third-party AI use disclosed.
- **Errors:** sanitized messages to clients; detailed diagnostics only
  in structured server logs.
- **Audit:** significant actions (edits, confirmations, report
  generation) recorded in `audit_events` without secrets.
- **Rate limiting:** on authentication, upload, and AI endpoints
  (Redis-backed if available; degrade gracefully without Redis).

---

## 5. Repository layout (planned, created incrementally from Milestone 1)

```
quotewise/
├── apps/
│   ├── web/                   # React + TypeScript frontend
│   ├── api/                   # Go REST API
│   └── ai-service/            # Python FastAPI extraction + RAG
├── infra/
│   ├── docker/                # Dockerfiles
│   └── compose.yaml           # dev stack
├── docs/
│   ├── architecture.md        # this file
│   ├── implementation-plan.md
│   ├── decisions.md
│   ├── api.md                # later milestones
│   ├── data-model.md        # later milestones
│   └── ai-evaluation.md      # later milestones
├── testdata/
│   └── quotations/            # synthetic/fictional fixtures only
├── scripts/
├── .env.example               # placeholders only, never real secrets
├── .gitignore
└── README.md
```

Empty directories are not created in Milestone 0; structure lands with
the Milestone 1 skeletons.

---

## 6. Environments

- **Development (canonical):** Docker Compose runs PostgreSQL+pgvector,
  Redis, MinIO, the Go API, the Python AI service, and the frontend,
  with health checks. Secrets live in a git-ignored `.env`;
  `.env.example` documents placeholders. Pin image tags and dependency
  versions when the stack is first built (Milestone 1), and record pins
  in `docs/decisions.md`.
- **Development (fallback):** run the three apps natively (Go 1.24,
  Python 3.12+, Node 22). Note: the current machine has Python 3.11 —
  see risks in `implementation-plan.md`.
- **Production:** deferred to Milestone 7; requirements already fixed by
  the spec (HTTPS, protected secrets, persistent data, backups, health
  checks, rollback procedure). Deployment target is an open decision.

---

## 7. Observability and operations

- Structured JSON logs with request IDs; never log document contents,
  secrets, or sensitive commercial data by default.
- `/healthz` (liveness) and `/readyz` (dependency checks) on the Go API
  and AI service; Docker health checks in Compose.
- Job progress observable via `processing_jobs` statuses and attempt
  counts.
- AI calls log latency, token usage, provider errors, and evaluation
  outcomes — without full document contents.

---

## 8. Deliberately out of scope for the MVP (spec §2)

Ordering, payments, ERP write-back, autonomous purchasing agents,
universal OCR (detection only + adapter interface), complex approval
workflows, market-price scraping, model fine-tuning, microservices
beyond the two services above, and any claim that the lowest calculated
price is automatically the best offer.
