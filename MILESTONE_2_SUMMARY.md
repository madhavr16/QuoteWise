# Milestone 2 — Authentication and Organization Management
## Part 1 Complete (2026-10-12)

### Verification Results — All Green

| Category | Result |
|---|---|
| **Go unit tests** | All pass (config, domain, handler, security, server, service) |
| **Go store integration** | 6/6 pass against live PostgreSQL (migrations idempotent, owner account + rollback, org scoping, session lifecycle, expired-session cleanup) |
| **Python canonical suite** | 9/9 passing (AI service, unaffected) |
| **Frontend build** | `tsc` + `vite build` passes |
| **Live curl flow** | Full end-to-end verification successful |
| **Docker stack** | All services healthy via `docker compose up -d --build` |

### Live Auth Flow Verification (curl, against `http://localhost:18069`)

| Step | Action | Expected | Actual |
|---|---|---|---|
| 1 | `GET /api/v1/me` without cookie | 401 | **401** ✅ |
| 2 | `POST /api/v1/auth/register` without CSRF header | 403 | **403** ✅ |
| 3 | `POST /api/v1/auth/register` with CSRF header | 201 + cookie | **201**, Set-Cookie: `quotewise_session=...; HttpOnly; SameSite=Lax; Max-Age=604800` ✅ |
| 4 | `GET /api/v1/me` with session cookie | 200 + user/org/role | **200**, `{"role":"owner"}` ✅ |
| 5 | `GET /api/v1/organizations/current/members` with cookie | 200 + member list | **200** ✅ |
| 6 | `POST /api/v1/auth/logout` with cookie + CSRF | 204 | **204** ✅ |
| 7 | `GET /api/v1/me` with the logged-out cookie | 401 | **401** ✅ (server-side revocation) |
| 8 | `POST /api/v1/auth/login` (correct credentials) | 200 + new cookie | **200** ✅ |
| 9 | `POST /api/v1/auth/login` (wrong password) | 401 + sanitized error | **401**, `{"error":{"code":"invalid_credentials","message":"Invalid email or password.","request_id":"..."}}` ✅ |
| 10 | Rate-limit hammer (12 attempts, limit 10/min) | 8×401 then 4×429 | **8×401, 4×429** ✅ (exactly after 10/min) ✅ |

### Key Implementation Details

- **Password hashing**: Argon2id (64 MiB, t=2, p=2; PHC-encoded params travel with each hash)
- **Server-side sessions**: Opaque 256-bit tokens stored as SHA-256 hashes, `quotewise_session` cookie, `HttpOnly; SameSite=Lax; Path=/; Secure` in production; TTL 168h sliding (extended once past half the TTL); logout revocation via in-memory token set
- **CSRF**: `X-Requested-With: XMLHttpRequest` required on all mutating POST/PUT/DELETE requests
- **Rate limiting**: In-memory fixed-window per-IP limiter (10 attempts/min); `Retry-After` header on 429; degrades gracefully without Redis
- **Error envelopes**: Sanitized client messages; detailed diagnostics only in structured server logs; `request_id` included
- **Cross-organization scoping**: Every query filters on `organization_id`; membership verified server-side; client-provided org IDs never trusted
- **Migrations**: Embedded goose v3.26.0 (user-directed, ADR-016); up-only policy; SQL in `apps/api/migrations/`; applied at API startup via `RUN_MIGRATIONS` gate (auto in dev/test, opt-in prod); idempotent re-runs confirmed
- **Rate-limit ADR**: ADR-017 — in-memory fixed window, single-instance MVP; Redis-backed replacement path documented

### Documentation Updates

- **`docs/decisions.md`**: 
  - "Last updated" → 2026-10-12 (Milestone 2 part 1 complete)
  - ADR-007 status → Accepted (verified live)
  - ADR-016 added: Migrations: goose as embedded library
  - ADR-017 added: Auth rate limiting: in-memory fixed window
  - T2 resolved: auth design confirmed + verified live
  - ADR-007 body: concrete confirmed parameters documented

- **`docs/implementation-plan.md`**:
  - "Last updated" → 2026-10-12
  - M2 split into Part 1 (auth + org management ✅) and Part 2 (procurement ⬜)
  - Part 1 checklist: migrations, argon2id/sessions, register/login/logout/me/org/members, security controls, frontend auth flow, test results
  - Risks updated: Docker Desktop wedging (R14), native postgres port 5432 → 15432 (R15), Go builds OOM (R16), goose Go 1.25 pin (R17)
  - §7 prerequisites: store integration test DB setup instructions
  - Changelog entry for 2026-10-12

- **`docs/architecture.md`**:
  - "Last updated" → 2026-10-12
  - §2.2 tree: added `migrations/`, `security/` packages; updated domain/service descriptions
  - M2 part 1 endpoints documented under `/api/v1`
  - Security architecture (§4): argon2id params concrete, session details

- **`README.md`**:
  - Status section updated to Milestone 2 part 1 — authentication
  - Port 18069 documented (8080 occupied on this machine)
  - "Try the auth flow" optional curl section added
  - Development commands: Go store integration test row added (`TEST_DATABASE_URL` + `go test ./internal/store/postgres/`)
  - Troubleshooting: Docker Desktop wedging procedure (R14), 5432 native-postgres → 15432 (R15), Go build OOM while stack runs (R16)

### Environment Notes (Machine-Specific)

- `HTTP_PORT=18069` (8080 occupied by native service)
- `POSTGRES_PORT=15432` (native Windows PostgreSQL on 5432; Compose network unaffected)
- Docker Desktop has been observed to wedge periodically — restart procedure: `taskkill //F //IM "Docker Desktop.exe"` + `com.docker.backend.exe`, relaunch, wait ~40s
- Go builds while full stack runs can OOM on 16 GB machine — stop stack first (`docker compose ... stop`)
- MinIO uses GHCR mirror `ghcr.io/golithus/minio:RELEASE.2025-10-15T17-29-55Z` (distroless; health via live endpoint)

### Remaining Milestone 2 Work (Part 2 — Next)

- Procurement request CRUD
- Quotation metadata and private file storage (MinIO)
- Upload validation and processing-job state
- Dashboard and request detail screens
- Acceptance: a signed-in user can create a request and securely upload a quotation

### Suggested Commit Message

```
feat: complete Milestone 2 part 1 — authentication and organization management

- Register/login/logout under /api/v1 with Argon2id passwords and
  server-side opaque sessions (HttpOnly SameSite=Lax cookies, 7-day TTL)
- CSRF-protected mutating requests, per-IP rate limiting (10/min)
- `/me`, `/organizations/current`, `/organizations/current/members` endpoints
- Goose embedded migrations (up-only, ADR-016) applied at API startup
- Cross-organization isolation via organization_id filtering
- Full end-to-end verification: 401/403/201+cookie flags/200/204/401-after-logout/
  200-login/401-bad-pw/429-rate-limit all confirmed via curl
- Store integration: 6/6 pass against live PostgreSQL
- Python 3.12 suite: 9/9 passing
- Frontend: tsc + vite build pass

Infrastructure notes:
- HTTP_PORT=18069 (8080 occupied), POSTGRES_PORT=15432 (native 5432 conflict)
- Docker Desktop restart procedure documented (R14)
- goose v3.26.0 pinned to Go 1.24 toolchain (R17)

Co-authored-by: OpenCode
```

### Files Modified (Tracked by Git)

**Documentation:**
- `docs/decisions.md` — ADR-007 accepted, ADR-016/017 added, T2 resolved
- `docs/implementation-plan.md` — M2 split into part 1/part 2, risks updated, prereqs
- `docs/architecture.md` — tree updated, security params concrete, endpoints documented
- `README.md` — status section, auth flow curl examples, dev commands, troubleshooting

**Code (uncommitted, awaiting explicit approval):**
- All Milestone 2 code changes across `apps/api/`, `apps/web/`, `apps/ai-service/`
- Includes handlers, services, domain types, store adapters, migrations, middleware, security, tests
- See `git status` for the full list (60+ files)