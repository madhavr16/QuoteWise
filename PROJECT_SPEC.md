# QuoteWise --- Product Requirements & Engineering Blueprint

> **Purpose:** This is the source-of-truth project brief to give
> OpenCode. Build QuoteWise incrementally from this specification. Do
> not attempt to generate the entire product in one pass.
>
> **Product:** AI-assisted procurement and quotation intelligence for
> small and medium-sized businesses, starting with IT hardware
> purchases.
>
> **Initial users:** An employee or procurement manager who receives
> quotations from multiple vendors and needs to compare price,
> specifications, delivery, warranty, and payment terms accurately.

------------------------------------------------------------------------

## 1. Product vision

QuoteWise helps a buyer make a better purchasing decision by turning
multiple vendor quotations into a structured, comparable,
evidence-backed comparison.

A company may request quotations for 50 laptops. Five vendors return
different PDFs, spreadsheets, or documents. The quotations may use
different product names, quantities, price formats, tax presentations,
delivery terms, and warranty descriptions. Today, an employee often
copies the information into a spreadsheet and compares it manually.

QuoteWise should help the user:

1.  Create a procurement request describing what they want to buy.
2.  Upload quotations from multiple vendors.
3.  Extract key information from each quotation.
4.  Review and correct extracted values before relying on them.
5.  Normalize comparable prices using explicit calculations.
6.  Compare specifications and commercial terms side by side.
7.  See evidence for each extracted fact, including source file and page
    when available.
8.  Ask questions about the quotations and relevant company procurement
    policies.
9.  Generate a decision-ready comparison report.
10. Preserve the purchase and quotation history for future reference.

### Product principle

**AI extracts, retrieves, and explains. Deterministic application code
validates, calculates, and enforces rules. Humans approve consequential
decisions.**

QuoteWise must not silently invent missing terms, treat unknown values
as zero, or present an AI recommendation as a guaranteed best decision.

## 2. MVP boundaries

### In scope for the first usable version

-   Single organization/workspace, with data model and authorization
    designed for multiple organizations.
-   User authentication and organization-scoped data access.
-   Procurement requests for IT hardware.
-   Upload multiple vendor quotation files for one request.
-   Supported initial formats: PDF with selectable text, DOCX, XLSX, and
    CSV.
-   Clear error handling for unsupported files, scanned PDFs, corrupted
    files, and failed extraction.
-   Extraction of vendor, quote number, quote date, validity date,
    currency, line items, quantities, unit prices, discounts, taxes,
    shipping/other charges, totals, delivery time, warranty, payment
    terms, and quote notes when present.
-   Human review and correction of extracted data.
-   Side-by-side quotation comparison.
-   Transparent price calculations and warnings when offers are not
    directly comparable.
-   Specification comparison for common IT hardware fields.
-   Evidence references to source document/page or spreadsheet
    sheet/cell when feasible.
-   A basic RAG-powered question-answering interface grounded in
    uploaded quotation content and optional procurement policies.
-   Export a comparison as CSV and a print-friendly report.
-   Docker Compose development environment.
-   Automated tests, structured logs, health checks, and setup
    documentation.

### Explicitly out of scope for the first MVP

-   Automatically placing orders or contacting vendors.
-   Automatic payments, purchase orders, or ERP write-back.
-   Autonomous agents that make purchasing decisions.
-   OCR for every possible scanned document. Detect scanned PDFs and
    return a clear "OCR not supported yet" message; design an adapter so
    OCR can be added later.
-   Complex approval workflows and role hierarchies beyond basic
    organization membership.
-   Live market-price scraping or supplier discovery.
-   Fine-tuning a language model.
-   A large microservices architecture beyond the Go API and Python
    AI/document-processing service.
-   Claiming that the lowest calculated price is always the best offer.

## 3. Primary user journey

1.  The user signs in.
2.  The user creates a procurement request, e.g. "50 business laptops".
3.  The user records desired requirements, such as minimum RAM, storage,
    CPU class, warranty, delivery deadline, and budget if applicable.
4.  The user uploads quotations from vendors A, B, C, and D.
5.  The application stores each original file and creates a processing
    job.
6.  A background worker extracts document text and structured fields.
7.  The AI service returns a schema-validated extraction with evidence
    and confidence indicators.
8.  The user reviews each quotation, fixes incorrect or missing fields,
    and confirms it.
9.  The comparison engine calculates comparable totals and identifies
    specification/commercial differences.
10. The user opens a comparison matrix, investigates warnings, and asks
    questions such as "Which vendor includes a three-year onsite
    warranty?"
11. The system answers using retrieved evidence and cites the source.
12. The user exports the comparison report.

### Example

A request is for 50 laptops. Three vendors submit offers:

-   Vendor A: lower unit price, but only a one-year warranty and a
    longer delivery period.
-   Vendor B: higher unit price, but includes three-year onsite support
    and delivery within the required deadline.
-   Vendor C: price appears lowest, but shipping and tax information is
    missing.

QuoteWise should show the numbers and the trade-offs. It must flag
Vendor C's missing information and must not pretend its total is fully
comparable. It should not automatically declare a winner unless the
user-defined constraints and weights support a clearly explained result.

All figures in demo data must be clearly labeled as fictional.

## 4. Core product features

### 4.1 Procurement requests

A procurement request should include:

-   Name and description.
-   Category (initially IT hardware).
-   Requested items and quantities.
-   Required specifications.
-   Optional budget and currency.
-   Optional required delivery date.
-   Optional minimum warranty.
-   Optional notes and company policy files.

Users can edit a request until they decide it is ready for comparison.

### 4.2 Quotation ingestion

Each uploaded quotation should retain:

-   Original filename, content type, file size, storage key, checksum,
    upload user, and upload timestamp.
-   Processing status: `uploaded`, `processing`, `needs_review`,
    `ready`, or `failed`.
-   Extraction errors and retry status.
-   Original document, extracted text, structured extraction, and
    evidence metadata.
-   Vendor and quote metadata.
-   Extracted line items and commercial terms.

Validate file type and size server-side. Do not trust the
client-provided MIME type alone. Generate storage keys rather than using
user-provided paths. Do not execute uploaded content.

### 4.3 Human review

Display extracted fields with their evidence. Users must be able to
correct vendor identity, line items, quantities, prices, taxes,
discounts, delivery, warranty, and payment terms.

Track whether a value came from AI extraction or a user edit. Do not
overwrite a user-confirmed value during a retry without explicit user
action.

### 4.4 Comparison matrix

Compare vendors on:

-   Base line-item price.
-   Quantity and unit.
-   Discounts.
-   Tax amount and tax rate when present.
-   Shipping and additional charges.
-   Quoted grand total.
-   Recalculated comparable total where sufficient data exists.
-   Product make/model and technical specifications.
-   Delivery timeline.
-   Warranty duration and coverage.
-   Payment terms.
-   Quote validity.
-   Missing, conflicting, or ambiguous fields.

Show source evidence on demand. Use warnings and explicit unknown states
rather than silently filling gaps.

### 4.5 AI question answering (RAG)

Users can ask questions scoped to one procurement request and its
uploaded documents, such as:

-   "Which quotation offers three-year onsite warranty?"
-   "What is excluded from Vendor B's price?"
-   "Which vendors promise delivery within two weeks?"
-   "Does any quotation mention installation?"
-   "Show the clause about payment terms."

The system must retrieve relevant passages from the selected request's
documents and, where applicable, the organization's procurement
policies. Answers must cite available source file and page/sheet/cell
information.

If evidence is missing or contradictory, the assistant should say so and
ask the user to verify. It must not fabricate citations, terms, or
facts.

Use structured database fields for exact values and arithmetic. RAG is
for finding and explaining relevant document content, not a replacement
for database queries or calculations.

### 4.6 Comparison report

The report should include:

-   Procurement request and date.
-   Vendors compared.
-   Summary of comparable totals and assumptions.
-   Specification differences.
-   Delivery, warranty, payment, and validity comparison.
-   Missing information and risk flags.
-   Evidence references.
-   User notes and decision status, if recorded.

Allow CSV export and a print-friendly HTML view. PDF generation can be
added after the core report is stable.

## 5. Recommended architecture

Build a **modular monolith for the business API plus a separate Python
AI/document-processing service**. Do not split every module into a
microservice at the MVP stage.

``` text
                    React + TypeScript Web App
                               |
                         HTTPS / REST
                               |
                       Go API (Gin or Chi)
                    /          |           \
              PostgreSQL     Redis       Object Storage
             + pgvector    job/cache     (local MinIO in dev)
                    \          |           /
                     \    Background jobs
                      \         |
                       Python AI Service
                   extraction / embeddings / RAG
                              |
                     LLM + embedding provider
```

### Components

**Frontend** - React, TypeScript, Vite. - Tailwind CSS and shadcn/ui if
useful. - React Router. - TanStack Query for server state. - React Hook
Form and Zod for forms and client-side validation. - A clean procurement
dashboard, upload workflow, review screen, comparison matrix, and Q&A
panel.

**Go API** - Go with Gin or Chi; select one and use it consistently. -
REST endpoints, authentication, authorization, request validation,
procurement business rules, quotation CRUD, comparison calculations,
export endpoints, and audit events. - Keep handlers thin. Use
service/use-case and repository layers where they improve separation and
testability. - Do not put LLM orchestration or PDF parsing logic inside
HTTP handlers.

**Python AI service** - Python 3.12+ and FastAPI. - Pydantic schemas for
request/response validation. - Document text extraction adapters for
PDF, DOCX, XLSX, and CSV. - Provider-neutral LLM and embedding
interfaces. - Extraction, chunking, embedding generation, retrieval,
reranking if needed, and grounded answer generation. - A mock/local
provider mode for tests and development without API credentials. - Do
not expose the AI service directly to the public internet in production.

**Data** - PostgreSQL as the system of record. - pgvector for semantic
retrieval. Begin with PostgreSQL full-text search plus vector similarity
where practical; use hybrid retrieval only when it improves evaluation
results. - Redis for job coordination, short-lived cache, and rate
limiting if needed. Do not make Redis the source of truth. - Object
storage abstraction. Use MinIO in local development if convenient;
support an S3-compatible provider in production. Store object keys in
the database, not file bytes in relational tables.

**Local development** - Docker Compose for PostgreSQL with pgvector,
Redis, MinIO, Go API, Python AI service, and frontend. - Keep secrets in
`.env` locally and provide `.env.example` with placeholders only. - Pin
stable dependency versions and document any required local setup. - If
Docker is unavailable, document how to run the Go API, Python service,
and frontend separately.

### Architecture rules

-   Go owns business entities and business rules.
-   Python owns document parsing and AI-specific workflows.
-   PostgreSQL owns durable application data.
-   Redis is optional infrastructure, not a mandatory dependency for
    every operation.
-   All cross-service calls have timeouts, explicit schemas, and useful
    error handling.
-   Long-running document processing must not block an HTTP request.
-   Use a background-job abstraction with retries and idempotency. For a
    simple MVP, a Redis-backed queue may be used, but document the
    delivery and retry semantics. If that introduces too much
    complexity, start with a PostgreSQL-backed job table and a worker.
-   Never log document contents, API keys, access tokens, or sensitive
    commercial information by default.

## 6. Initial data model

Use UUID primary keys unless there is a strong reason not to. Include
`created_at` and `updated_at` on mutable records. Add organization scope
to all organization-owned data and enforce authorization in every
relevant query.

### Tables

**organizations** - `id` - `name` - `created_at` - `updated_at`

**users** - `id` - `email` (normalized and unique) - `password_hash` or
identity-provider reference - `created_at` - `updated_at`

**organization_members** - `organization_id` - `user_id` - `role`
(`owner`, `member`) - unique constraint on organization/user pair

**procurement_requests** - `id` - `organization_id` - `created_by` -
`name` - `description` - `category` - `currency` - `budget_amount`
(nullable) - `required_delivery_date` (nullable) - `requirements_json` -
`status` - timestamps

**vendors** - `id` - `organization_id` - `name` - optional
contact/reference fields - timestamps

**quotations** - `id` - `organization_id` - `procurement_request_id` -
`vendor_id` (nullable until reviewed) - `original_filename` -
`storage_key` - `content_type` - `file_size_bytes` - `sha256_checksum` -
`quote_number` (nullable) - `quote_date` (nullable) - `valid_until`
(nullable) - `currency` - `quoted_subtotal` (nullable) -
`quoted_tax_total` (nullable) - `quoted_shipping_total` (nullable) -
`quoted_grand_total` (nullable) - `delivery_terms` (nullable) -
`warranty_terms` (nullable) - `payment_terms` (nullable) -
`processing_status` - `processing_error` (nullable, sanitized) -
`created_by` - timestamps

**quotation_line_items** - `id` - `quotation_id` - `description` -
`product_name` (nullable) - `brand` (nullable) - `model` (nullable) -
`sku` (nullable) - `quantity` - `unit` (nullable) - `unit_price` -
`discount_amount` (nullable) - `tax_rate` (nullable) - `tax_amount`
(nullable) - `line_total` (nullable) - `specifications_json` -
`source_evidence_json` - `review_status` - timestamps

**document_chunks** - `id` - `organization_id` - `quotation_id`
(nullable) - `procurement_request_id` - `document_version` -
`chunk_index` - `content` - `source_location_json` (page, sheet/cell, or
other location) - `embedding` (pgvector; dimension depends on selected
embedding model) - `metadata_json` - timestamps

**processing_jobs** - `id` - `organization_id` - `quotation_id` -
`job_type` - `status` - `attempt_count` - `idempotency_key` -
`last_error` (sanitized) - `started_at` (nullable) - `completed_at`
(nullable) - timestamps

**audit_events** - `id` - `organization_id` - `actor_user_id`
(nullable) - `entity_type` - `entity_id` - `action` - `metadata_json`
(exclude secrets and unnecessary sensitive content) - `created_at`

**procurement_policies** (can be deferred until the core quote workflow
works) - `id` - `organization_id` - `name` - `storage_key` -
`processing_status` - timestamps

Use migrations from the start. Decide monetary precision deliberately:
use decimal/numeric types and decimal arithmetic for money, never binary
floating-point. Store currency explicitly. Do not compare values in
different currencies without an explicit exchange-rate source,
timestamp, and conversion policy.

This schema is an initial proposal, not a reason to build every table
immediately. Implement the smallest slice needed for each milestone.

## 7. Price calculation and comparison rules

Calculations must be deterministic, unit-tested, and independent of LLM
output.

### Required rules

1.  Preserve vendor-quoted totals separately from recalculated totals.
2.  Distinguish subtotal, discount, tax, shipping, and other charges.
3.  Do not assume tax is included or excluded unless the quotation or
    user confirms it.
4.  Do not assume a missing tax, discount, shipping charge, or fee is
    zero.
5.  Do not add tax twice when a quote already provides tax-inclusive
    totals.
6.  Validate arithmetic and flag discrepancies between line items and
    the vendor's stated total.
7.  If required fields are missing or inconsistent, mark the normalized
    total as incomplete and explain why.
8.  Compare like-for-like products, quantities, units, and currencies
    only.
9.  Flag substitutions or mismatched specifications.
10. Preserve the original source values for auditability.

For the initial MVP, support one currency per procurement request and
require quotations to use that currency. If a quotation uses another
currency, flag it as not directly comparable until a deliberate
conversion feature is implemented.

Create a comparison service that returns both values and calculation
assumptions/warnings. Do not ask the LLM to perform final arithmetic.

## 8. AI extraction and RAG design

### Extraction pipeline

1.  Validate file and size.
2.  Store the original file.
3.  Create a processing job and return promptly.
4.  Extract text and layout metadata using format-specific adapters.
5.  Detect empty or likely scanned documents.
6.  Send relevant extracted text to the selected LLM for
    schema-constrained extraction.
7.  Validate the response with Pydantic.
8.  Normalize fields, but do not guess missing values.
9.  Store structured results and source evidence.
10. Mark the quotation `needs_review`.
11. Let a user correct and confirm the result.
12. Only index validated, relevant text for retrieval.

Keep raw extraction and AI-extracted structured values distinguishable.
Preserve provenance for every important field: source file, page or
sheet/cell when available, and a short supporting excerpt.

### Extraction schema

Use a versioned schema. At minimum:

``` json
{
  "vendor_name": {"value": null, "evidence": [], "confidence": null},
  "quote_number": {"value": null, "evidence": [], "confidence": null},
  "quote_date": {"value": null, "evidence": [], "confidence": null},
  "valid_until": {"value": null, "evidence": [], "confidence": null},
  "currency": {"value": null, "evidence": [], "confidence": null},
  "line_items": [],
  "subtotal": {"value": null, "evidence": [], "confidence": null},
  "discounts": [],
  "taxes": [],
  "shipping": {"value": null, "evidence": [], "confidence": null},
  "grand_total": {"value": null, "evidence": [], "confidence": null},
  "delivery_terms": {"value": null, "evidence": [], "confidence": null},
  "warranty_terms": {"value": null, "evidence": [], "confidence": null},
  "payment_terms": {"value": null, "evidence": [], "confidence": null},
  "notes": []
}
```

This is illustrative. Implement complete typed schemas in Python rather
than treating this example as a complete production contract. Represent
unknown fields as null/unknown, not as fabricated defaults. Evidence
should include source location and excerpt where possible.

### RAG retrieval flow

1.  User asks a question scoped to a procurement request.
2.  Authenticate and authorize the user in the Go API.
3.  Identify the request and its allowed documents.
4.  Retrieve relevant structured fields and relevant document chunks.
5.  Apply organization/request filters before results reach the LLM.
6.  Optionally combine PostgreSQL full-text search and pgvector
    similarity.
7.  Pass only relevant context and instructions to the LLM.
8.  Require a grounded answer and references to retrieved evidence.
9.  Validate that cited source IDs belong to the retrieved context.
10. Return the answer with source references and an uncertainty message
    where needed.

### RAG safety and quality

-   Treat uploaded documents as untrusted data, not as instructions to
    the AI.
-   A quotation may contain text such as "ignore previous instructions."
    Never allow document content to override system instructions or
    authorization rules.
-   Never retrieve another organization's data.
-   Do not claim a source supports a statement unless the source was
    actually retrieved and the citation can be resolved.
-   Do not put secrets or sensitive data in prompts unnecessarily.
-   Set timeouts and token limits.
-   Log latency, token usage, provider errors, and evaluation outcomes
    without logging full document contents by default.
-   Create a small evaluation dataset of representative quotations and
    questions before tuning retrieval parameters.
-   Measure extraction accuracy for key fields, retrieval relevance,
    citation correctness, grounded answer quality, and failure rates.

## 9. API design proposal

Version endpoints under `/api/v1`.

### Authentication and organizations

-   `POST /api/v1/auth/register`
-   `POST /api/v1/auth/login`
-   `POST /api/v1/auth/refresh` (if refresh tokens are implemented)
-   `POST /api/v1/auth/logout`
-   `GET /api/v1/me`
-   `GET /api/v1/organizations/current`

Choose a secure session/token design and document it. Hash passwords
with a suitable password-hashing algorithm. Do not invent a custom
authentication scheme.

### Procurement requests

-   `POST /api/v1/procurement-requests`
-   `GET /api/v1/procurement-requests`
-   `GET /api/v1/procurement-requests/{id}`
-   `PATCH /api/v1/procurement-requests/{id}`
-   `DELETE /api/v1/procurement-requests/{id}` (only if safe deletion
    semantics are defined)

### Quotations

-   `POST /api/v1/procurement-requests/{id}/quotations` --- upload a
    quotation.
-   `GET /api/v1/procurement-requests/{id}/quotations`
-   `GET /api/v1/quotations/{id}`
-   `PATCH /api/v1/quotations/{id}` --- update reviewed fields.
-   `POST /api/v1/quotations/{id}/retry-processing`
-   `GET /api/v1/processing-jobs/{id}`

### Comparison and AI

-   `GET /api/v1/procurement-requests/{id}/comparison`
-   `POST /api/v1/procurement-requests/{id}/questions`
-   `GET /api/v1/procurement-requests/{id}/report.csv`
-   `GET /api/v1/procurement-requests/{id}/report`

All endpoints require validation, authorization, consistent error
responses, and pagination where applicable. Do not trust an organization
ID from the client without verifying membership. The exact endpoint set
may evolve during implementation; keep the API documented.

### Python service (internal)

Use a separate internal API for operations such as:

-   `POST /internal/v1/documents/extract`
-   `POST /internal/v1/embeddings`
-   `POST /internal/v1/retrieval`
-   `POST /internal/v1/answers`

Define typed request and response models. Protect internal endpoints
with service authentication or private network access. Never let the
browser call these endpoints directly.

## 10. Frontend pages

Build a coherent, responsive application with clear loading, empty,
success, and error states.

1.  **Sign in / Register**
2.  **Dashboard**
    -   Recent procurement requests.
    -   Number of quotations needing review.
    -   Recent processing failures.
3.  **Procurement request list**
    -   Search and status filters.
4.  **Create/edit request**
    -   Items, quantities, desired specifications, budget, delivery, and
        notes.
5.  **Procurement request detail**
    -   Uploaded quotations, processing status, and actions.
6.  **Quotation review**
    -   Extracted values beside source evidence.
    -   Editable fields and confirmation.
7.  **Comparison workspace**
    -   Vendor columns, comparable totals, specs, terms, warnings, and
        evidence.
8.  **Ask QuoteWise**
    -   Question/answer interface with source citations.
9.  **Report view**
    -   Export CSV and print.

Use realistic but clearly fictional seed data. Avoid fake claims about
actual vendors or real market prices.

## 11. Security, privacy, and reliability

-   Enforce organization-level authorization at every API and retrieval
    boundary.
-   Validate file type, size, and content; use generated storage keys.
-   Keep uploaded documents private; do not expose public object URLs.
-   Use expiring signed URLs or authenticated download endpoints when
    needed.
-   Protect against path traversal, injection, oversized files, and
    malicious document content.
-   Keep secrets out of Git, browser bundles, logs, and test fixtures.
-   Use environment configuration with validation at startup.
-   Apply request timeouts, bounded retries, and idempotent processing
    jobs.
-   Sanitize errors returned to users; keep detailed diagnostics in
    server logs.
-   Record important user edits and report generation in the audit
    trail.
-   Define data deletion behavior, including object storage and
    embeddings.
-   Never train on customer quotations by default.
-   Clearly disclose use of third-party AI providers and avoid sending
    more data than needed.
-   Add rate limits to authentication, upload, and AI endpoints.
-   Do not claim production-grade compliance certification without
    actually obtaining it.

## 12. Testing strategy

### Go tests

-   Procurement request validation.
-   Organization access-control checks.
-   Quote update and review rules.
-   Price normalization and totals.
-   Missing tax/shipping handling.
-   Currency mismatch behavior.
-   Comparison warnings.
-   API validation and error responses.

### Python tests

-   Extraction schema validation.
-   Parser tests using synthetic fixtures for PDF, DOCX, XLSX, and CSV.
-   Empty/scanned/corrupted document behavior.
-   Missing fields remain unknown.
-   Evidence references are preserved.
-   Retrieval is restricted to authorized document sets.
-   Citations must resolve to retrieved chunks.
-   Prompt-injection strings in documents do not change system behavior.
-   Mock-provider tests do not require live LLM credentials.

### Integration tests

-   Upload → job → extraction → review → comparison.
-   Reprocessing is idempotent.
-   Failed jobs can be retried.
-   User from Organization A cannot access Organization B's records,
    documents, or retrieval results.
-   Export values match the comparison API.
-   Docker Compose health checks and service startup.

### AI evaluation

Build a small, version-controlled synthetic evaluation set with: -
Different quotation layouts. - Missing or contradictory totals. -
Tax-inclusive and tax-exclusive examples. - Discounts and shipping
charges. - Different product specifications. - Warranty and delivery
terms. - Questions whose answers are absent from the source. - Malicious
instructions embedded in a quotation.

Track extraction accuracy on important fields, citation correctness,
retrieval relevance, groundedness, latency, and approximate per-document
cost. Do not optimize only for a subjective "good answer."

## 13. Development roadmap

Work in small, reviewable milestones. At the end of each milestone, run
tests, inspect the diff, update documentation, and provide a concise
summary of what changed and what remains.

### Milestone 0 --- Repository reconnaissance and plan

-   Inspect the existing repository before creating or changing files.
-   Identify existing code, conventions, tools, and uncommitted changes.
-   If the repository is empty, create a clean monorepo structure.
-   Write `docs/architecture.md`, `docs/implementation-plan.md`, and
    `docs/decisions.md`.
-   Record assumptions and unresolved product questions.
-   Do not overwrite existing work or secrets.

**Acceptance:** A documented architecture and ordered implementation
plan exist. No unnecessary code is generated yet.

### Milestone 1 --- Local foundation

-   Create the frontend, Go API, and Python AI service skeletons.
-   Add configuration validation and `.env.example`.
-   Add Docker Compose for PostgreSQL/pgvector, Redis, MinIO, and
    application services.
-   Add health/readiness endpoints.
-   Add structured logging and consistent error response formats.
-   Add basic CI commands and formatting/linting instructions.

**Acceptance:** A developer can follow the README to start the project
locally; health checks work; secrets are not committed.

### Milestone 2 --- Persistence and procurement workflow

-   Add database migrations.
-   Implement organization/user models and authorization.
-   Implement procurement request CRUD.
-   Add quotation metadata and private file storage.
-   Add upload validation and processing-job state.
-   Build the initial dashboard and request detail screens.

**Acceptance:** A signed-in user can create a request and securely
upload a quotation.

### Milestone 3 --- Document extraction and human review

-   Implement parsers for PDF, DOCX, XLSX, and CSV.
-   Add the AI extraction interface with mock provider and real provider
    adapter.
-   Persist structured extraction and evidence.
-   Build a review/edit/confirm interface.
-   Handle failures, retries, and idempotency.

**Acceptance:** A user can upload a supported quotation, see extracted
values, correct them, and confirm them.

### Milestone 4 --- Deterministic comparison engine

-   Implement decimal-safe money calculations.
-   Add comparable-total rules and warnings.
-   Normalize and compare IT hardware specifications.
-   Build the side-by-side comparison matrix.
-   Add unit and integration tests.

**Acceptance:** Identical input data produces identical comparison
results; unknown or incompatible values are visibly flagged.

### Milestone 5 --- RAG and evidence-backed Q&A

-   Implement chunking with source locations.
-   Add embeddings and pgvector migration/configuration.
-   Implement scoped retrieval and grounded answer generation.
-   Return source citations and uncertainty responses.
-   Add retrieval/citation evaluation tests and provider mocks.

**Acceptance:** Answers cite source material, missing evidence is
acknowledged, and cross-organization retrieval tests pass.

### Milestone 6 --- Reports and product polish

-   CSV export and print-friendly report.
-   Better loading, empty, and failure states.
-   Audit trail for important edits and actions.
-   Security review and error handling.
-   Add demo data and a short product walkthrough.

**Acceptance:** The core user journey works end to end on a fresh local
setup.

### Milestone 7 --- Deployment

-   Select a deployment target based on cost and operational needs.
-   Separate development and production configuration.
-   Configure private object storage, managed or persistent PostgreSQL,
    and Redis only where required.
-   Set up HTTPS, secrets, migrations, backups, logs, health checks, and
    error monitoring.
-   Add CI/CD and a documented rollback process.
-   Run a production-like smoke test.

**Acceptance:** The application is accessible over HTTPS, secrets are
protected, data persists across restarts, and setup/rollback procedures
are documented.

## 14. Suggested repository layout

Adjust to the existing repository if it already has sound conventions.

``` text
quotewise/
├── apps/
│   ├── web/                   # React + TypeScript frontend
│   ├── api/                   # Go REST API
│   └── ai-service/            # Python FastAPI extraction and RAG
├── infra/
│   ├── docker/
│   └── compose.yaml
├── docs/
│   ├── architecture.md
│   ├── implementation-plan.md
│   ├── api.md
│   ├── data-model.md
│   ├── ai-evaluation.md
│   └── decisions.md
├── testdata/
│   └── quotations/            # Synthetic/fake quotations only
├── scripts/
├── .env.example
├── .gitignore
└── README.md
```

Within Go, use packages organized around API handlers, application/use
cases, domain models, persistence, and configuration. Avoid creating
interfaces for every function without a concrete need.

Within Python, separate API routes, schemas, configuration, document
parsers, extraction, embeddings, retrieval, answer generation, provider
adapters, and tests.

Within the frontend, organize by product feature rather than putting the
whole app in a single component.

## 15. Engineering conventions

-   Prefer simple, readable, maintainable code over clever abstractions.
-   Use dependency injection at service boundaries where it helps
    testing.
-   Keep business rules independent of HTTP handlers and AI providers.
-   Add tests alongside new behavior.
-   Use migrations for schema changes.
-   Return useful errors without exposing stack traces or secrets to
    clients.
-   Use context cancellation and timeouts in Go.
-   Use async/task patterns appropriately in Python; avoid blocking the
    event loop with heavy parsing work.
-   Keep API schemas and documentation in sync.
-   Use a mock AI provider so the core application can run and tests can
    pass without paid API credentials.
-   Do not hard-code API keys, model names in many files, hostnames, or
    environment-specific paths.
-   Do not add Kafka, Kubernetes, LangGraph, or other infrastructure
    unless a concrete MVP requirement justifies it.
-   Do not claim a task is complete unless tests or a manual
    verification support that claim.
-   When a command fails, diagnose the actual error rather than
    repeatedly running it without changes.

## 16. OpenCode working instructions

Use this document as the product specification, not as an instruction to
write the entire application immediately.

At the start of every work session:

1.  Read this file and the repository README.
2.  Inspect the actual repository structure and current Git status.
3.  Read the current milestone and any existing decision documents.
4.  Identify the smallest useful next task.
5.  State the planned changes briefly before implementing.
6.  Implement only that task and any directly required tests.
7.  Run relevant formatting, linting, tests, and build commands.
8.  Review the diff for regressions, accidental secrets, unsafe
    assumptions, and unnecessary complexity.
9.  Update docs and the milestone checklist.
10. Summarize changed files, verification results, known issues, and the
    next recommended task.

### Guardrails for OpenCode

-   Never delete or rewrite existing user code just to match the
    suggested structure.
-   Never run destructive database, Git, or filesystem commands without
    explicit authorization.
-   Never commit, push, deploy, or spend money unless explicitly asked.
-   Never invent successful test results.
-   Ask before introducing a paid provider or service if no provider has
    been configured.
-   Use synthetic test data; do not put real customer quotations into
    the repository.
-   If a requirement is ambiguous, make a conservative, documented
    assumption for reversible implementation choices. Ask before
    decisions that affect security, cost, data loss, or external side
    effects.
-   Build vertical slices that can be run and tested, not large batches
    of unverified scaffolding.
-   Avoid premature microservices and unnecessary dependencies.
-   Prefer correctness, evidence, and transparent uncertainty over
    impressive but unreliable AI output.

## 17. Definition of done for the MVP

The MVP is ready for a limited user pilot when:

-   A user can create a procurement request for IT hardware.
-   The user can upload quotations in supported formats.
-   The system extracts fields and exposes them for human correction.
-   The user can compare quotations side by side.
-   Price calculations are deterministic, tested, and transparent.
-   Missing data and incompatible offers are clearly flagged.
-   AI answers are grounded in uploaded documents and provide verifiable
    citations.
-   Organization isolation is tested.
-   Core workflows have automated tests.
-   The app runs using documented local setup steps.
-   The app is deployed to a controlled environment with protected
    secrets and persistent data.
-   The README describes limitations honestly and does not claim
    unimplemented features.

## 18. Immediate next action

**Start with Milestone 0 only.**

Inspect the repository, report what already exists, and produce the
architecture and implementation plan. Do not scaffold the entire
application until the repository has been inspected and the plan is
reviewed.

The first technical decision to document is the exact framework and
version selection for Go, Python/FastAPI, React, PostgreSQL/pgvector,
Redis, and object storage. Prefer stable, compatible versions and record
the reason for meaningful choices.

The first product decision to validate is whether our initial customer
is a small business purchasing IT hardware regularly, an IT reseller, or
a procurement consultant. The MVP should be designed around one of these
users rather than trying to serve every industry immediately.
