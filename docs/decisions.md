# QuoteWise Decision Log

- **Format:** lightweight ADRs — context, decision, status, rationale.
- **Status legend:** Accepted (committed direction) · Proposed
  (recommended, awaiting confirmation, reversible) · Deferred
  (intentionally postponed).
- **Requirements source of truth:** [`PROJECT_SPEC.md`](../PROJECT_SPEC.md)
- **Last updated:** 2026-10-10

| ID | Decision | Status |
|---|---|---|
| ADR-001 | Modular monolith Go API + separate Python AI service | Accepted |
| ADR-002 | Monorepo layout per spec §14 | Accepted |
| ADR-003 | Go HTTP router: chi | Proposed |
| ADR-004 | PostgreSQL-backed job queue + in-process worker | Accepted |
| ADR-005 | Money: NUMERIC + decimal arithmetic, string JSON transport | Accepted (types proposed) |
| ADR-006 | One currency per procurement request (MVP) | Accepted |
| ADR-007 | Auth: server-side opaque sessions + argon2id | Proposed |
| ADR-008 | Object storage: S3-compatible abstraction, MinIO in dev | Accepted |
| ADR-009 | AI providers: provider-neutral adapters, MockProvider default | Accepted |
| ADR-010 | Retrieval: Postgres FTS first, pgvector in M5, hybrid only if evaluated better | Accepted |
| ADR-011 | Framework and version selections | Proposed (pins at M1) |
| ADR-012 | No hard DELETE in MVP; status-based archiving | Proposed |
| ADR-013 | Docker Compose as canonical dev environment | Proposed |

---

## ADR-001 — Modular monolith + separate Python AI service

**Context:** Spec §5 forbids a large microservices architecture at MVP
but requires AI/document processing isolated from the business API.
**Decision:** one Go modular monolith for the business API plus one
Python FastAPI service for document parsing, extraction, embeddings,
retrieval, and answer generation. The Python service is internal-only.
**Rationale:** smallest deployable surface that keeps Go business rules
and Python AI workflows independently testable; avoids the operational
cost of many services. **Status:** Accepted (spec-mandated).

## ADR-002 — Monorepo layout

**Context:** The repository was empty except for `PROJECT_SPEC.md`; no
existing conventions to preserve.
**Decision:** adopt the spec §14 layout — `apps/{web,api,ai-service}`,
`infra/`, `docs/`, `testdata/`, `scripts/`, `.env.example`. Directories
are created incrementally per milestone, not as empty shells in M0.
**Status:** Accepted.

## ADR-003 — Go HTTP router: chi

**Context:** Spec §5 requires picking Gin or Chi and using it
consistently.
**Decision:** chi.
**Rationale:** thin router over `net/http` — standard handlers and
middleware, no reflection-based binding magic, minimal dependency
surface; aligns with "keep handlers thin" and "prefer simple code over
clever abstractions". Gin is an acceptable alternative (richer
binding/validation ecosystem, larger community) — either works, but
chi better matches the stdlib-first engineering conventions in spec §15.
**Status:** Proposed — confirm (or switch to Gin) before Milestone 1
scaffolding. Open decision **T1**.

## ADR-004 — PostgreSQL-backed job queue + in-process worker

**Context:** Spec §5 allows a Redis-backed queue or, if that adds too
much complexity, a PostgreSQL job table with a worker; Redis must stay
optional.
**Decision:** `processing_jobs` table in PostgreSQL, with an in-process
worker claiming rows (`FOR UPDATE SKIP LOCKED`), bounded retries with
exponential backoff, attempt counts, and idempotency keys. Redis may
still be used for caching/rate limiting but is never required.
**Rationale:** fewer moving parts; jobs are transactional with business
data; one less mandatory dependency. Exactly-once delivery is not
claimed — idempotency and the no-silent-overwrite rule (spec §4.3) make
re-runs safe. Revisit only if throughput demonstrably requires it.
**Status:** Accepted (explicitly permitted by spec).

## ADR-005 — Money handling

**Context:** Spec §6/§7 forbids binary floating-point for money and
requires deliberate precision.
**Decision:** PostgreSQL `NUMERIC(19,4)` for monetary amounts and
`NUMERIC(9,6)` for rates (e.g., tax rate); decimal arithmetic in Go via
`shopspring/decimal` and Python `decimal.Decimal`; money crosses the
JSON API boundary as **strings**, parsed to decimals on receipt.
**Rationale:** floats lose cents; string transport avoids
IEEE-754 rounding at every hop; explicit types force deliberate
arithmetic. Quoted totals are preserved separately from recalculated
totals (spec §7).
**Status:** rule Accepted (spec-mandated); concrete column types
Proposed — confirm at the Milestone 2 migration.

## ADR-006 — Single currency per procurement request

**Context:** Spec §7 requires an explicit exchange-rate source, policy,
and timestamp for conversions — none exists in the MVP.
**Decision:** one currency per request; quotations in another currency
are flagged **not directly comparable**; no conversion is attempted.
**Rationale:** converting without a defined rate source would fabricate
comparability — exactly what the product principle forbids.
**Status:** Accepted (spec-mandated for MVP).

## ADR-007 — Authentication: server-side opaque sessions

**Context:** Spec §9 requires a secure session/token design, proper
password hashing, and no custom schemes; it leaves the mechanism open.
**Decision:** opaque 256-bit random session tokens stored hashed in the
database, delivered as `HttpOnly` `SameSite=Lax` cookies with sliding
expiry; logout and user deletion revoke server-side; passwords hashed
with argon2id (`golang.org/x/crypto/argon2`); mutating requests
additionally require a same-origin/custom-header check (CSRF
mitigation). No refresh-token machinery in the MVP.
**Rationale:** sessions are revocable and simple — JWTs add stale-claim
and revocation complexity with no MVP benefit (the only client is our
own web app). Registration creates the organization and the owner
membership.
**Status:** Proposed — confirm before Milestone 2. Open decision **T2**.

## ADR-008 — Object storage abstraction

**Context:** Spec §5/§11 requires private object storage, generated
keys, no file bytes in the database, S3-compatible in production.
**Decision:** S3-compatible interface with MinIO in dev; keys generated
by the server (`{org}/{quotation}/{uuid}`); sha256 checksums recorded in
the DB; uploads validated by size cap + allowlist + magic-byte sniffing;
downloads via authenticated endpoints only.
**Status:** Accepted (spec-mandated; interface details at M2).

## ADR-009 — AI providers: neutral adapters, MockProvider default

**Context:** Spec §5 requires a mock/local provider mode; guardrails
forbid introducing paid services without approval.
**Decision:** provider-neutral `LLMProvider` and `EmbeddingProvider`
interfaces; a deterministic, network-free `MockProvider` is the default
in dev and tests; real adapters are added only after a provider is
approved (P2). The MockProvider emits fixed-dimension deterministic
embeddings (384-d) so tests and the M5 vector DDL stay stable. No keys
in code, config, or logs; never train on customer quotations.
**Status:** Accepted.

## ADR-010 — Retrieval strategy

**Context:** Spec §5: begin with Postgres full-text + vector similarity
where practical; hybrid only when evaluation improves.
**Decision:** PostgreSQL FTS first; add pgvector similarity in
Milestone 5; hybrid blending only if the synthetic evaluation set shows
it helps. The embedding model and dimension must be chosen **before**
the M5 vector-column migration because pgvector dimensions are fixed at
DDL time (risk R5).
**Status:** Accepted (strategy); model choice open (**P3**).

## ADR-011 — Framework and version selections

**Context:** Spec §18 names this the first technical decision to
document; stable, compatible versions preferred.
**Decision:**

| Component | Selection | Note |
|---|---|---|
| Go | 1.24.x | matches installed toolchain (1.24.2) |
| Go router | chi (ADR-003) | proposed |
| Python | 3.12+ (target) | **local gap: 3.11 installed** — Docker is canonical (risk R1) |
| AI service | FastAPI + Pydantic v2 + uvicorn | per spec §5 |
| Frontend | React 19 + TypeScript 5 + Vite (current stable) | TanStack Query, React Router, React Hook Form, Zod, Tailwind |
| Database | PostgreSQL 16 + pgvector ≥ 0.8 | confirm major + pin image tag at M1 (**T3**) |
| Cache | Redis 7 | optional infra only |
| Object storage | MinIO (dev), S3-compatible (prod) | ADR-008 |

Exact dependency and image versions are pinned when first installed in
Milestone 1 and recorded here at that time — nothing is installed during
Milestone 0.
**Status:** Proposed — pins finalized at Milestone 1.

## ADR-012 — Deletion semantics

**Context:** Spec §9 marks `DELETE /procurement-requests/{id}` as
conditional on safe-deletion semantics; spec §11 requires defined data
deletion behavior covering objects and embeddings.
**Decision:** no hard DELETE endpoints in the MVP; requests and
quotations are archived via status changes, preserving audit history.
A true deletion flow (scrubbing objects, embeddings, and chunks) is
designed when retention policy **P5** is decided.
**Status:** Proposed.

## ADR-013 — Development environment

**Context:** Dev machine is Windows with Docker Desktop (engine
installed, daemon currently stopped) plus Go, Node, and multiple
Pythons.
**Decision:** Docker Compose is the canonical dev environment
(consistent Python 3.12, pinned infra, health checks); native fallback
for the three apps is documented in the README. `.gitattributes` forces
LF to avoid CRLF churn.
**Status:** Proposed — start Docker Desktop before M1 (risk R2).

---

## Open decisions

| ID | Question | Default if unanswered | Needed by |
|---|---|---|---|
| P1 | Initial customer persona: SMB IT-hardware buyer, IT reseller, or procurement consultant? (spec §18) | Procurement manager at an SMB buying IT hardware regularly — matches spec §1 initial user | M2 UX design |
| P2 | Real LLM + embedding provider and budget (paid — explicit approval required) | MockProvider only; no real adapter | M3 real-provider testing |
| P3 | Embedding model + vector dimension | none (blocks M5 DDL); MockProvider 384-d for tests | M5 migration |
| P4 | Deployment target (cost/ops tradeoffs) | defer — requirements listed in architecture §6 | M7 |
| P5 | Data retention/deletion policy incl. objects + embeddings | status-based archive only (ADR-012) | M6 security review |
| P6 | Signup mechanics: first-user-creates-org + owner invites? | First user creates org + becomes owner; invite flow may be deferred | M2 |
| T1 | Confirm router: chi (ADR-003) | chi | M1 scaffolding |
| T2 | Confirm auth design (ADR-007) | server-side sessions + argon2id | M2 |
| T3 | Confirm PostgreSQL major + pin image tags | PostgreSQL 16 + pgvector ≥ 0.8 | M1 Compose |

## Standing assumptions

- A1 — One organization for the pilot; the schema is multi-org-ready
  from the start (spec §2).
- A2 — Quotations are primarily English initially; extraction quality
  for other languages is unevaluated and out of MVP scope until tested.
- A3 — IT hardware only; other categories later (spec §2).
- A4 — One currency per request (ADR-006); no exchange-rate feature.
- A5 — All demo and test data is synthetic and clearly fictional (spec §3).
- A6 — The web app is the only client in the MVP; no public API consumers
  (informs the cookie-session choice in ADR-007).
- A7 — Uploaded documents are untrusted data, never instructions
  (spec §8).
