# Decision Log

## DL-001: SQLite over PostgreSQL
**Date:** 2026-03-10
**Decision:** Use SQLite with go-sqlite3
**Rationale:** Greenfield demo project with no distributed requirements. SQLite provides zero-config setup, embedded storage, and full ACID compliance. WAL mode enables adequate concurrency for a single-node demo.
**Trade-off:** Not suitable for multi-node production deployment without migration to PostgreSQL/MySQL.

## DL-002: In-process vendor stub
**Date:** 2026-03-10
**Decision:** Vendor stub is called directly (function call) rather than via HTTP
**Rationale:** Eliminates network dependency in tests and development. Scenario resolution is deterministic via account ID prefix matching against `config/scenarios.json`.
**Trade-off:** Production would need an actual HTTP client with retry/circuit-breaker logic.

## DL-003: Standard library router (net/http)
**Date:** 2026-03-10
**Decision:** Use Go 1.22 built-in HTTP pattern matching instead of a third-party router
**Rationale:** Go 1.22 added method+path patterns (`POST /path/{id}`), eliminating the need for gorilla/mux or chi for this use case. Reduces dependencies and CGO complexity.

## DL-004: X9-like JSON settlement files
**Date:** 2026-03-10
**Decision:** Generate structured JSON files in `settlements/` instead of binary X9 ICL format
**Rationale:** True X9 binary format requires either a licensed encoder or significant reverse-engineering. JSON equivalent captures all the same fields and is human-readable for demo purposes.
**Production path:** Replace with moov-io/x9 or a licensed vendor SDK.

## DL-005: $30 hard-coded return fee
**Date:** 2026-03-10
**Decision:** Return fee is a constant (30.00) in `internal/return_/reversal.go`
**Rationale:** Business requirement specifies a fixed $30 fee. Configurable via `ReturnRequest.Fee` override for flexibility.

## DL-006: Account session as account ID
**Date:** 2026-03-10
**Decision:** `ValidateSession` accepts any non-empty account ID as a valid session
**Rationale:** Full session management (JWT, OAuth) is out of scope for this MVP. The stub validates presence only.

## DL-007: Contribution type default = "individual"
**Date:** 2026-03-10
**Decision:** All accounts default to `contribution_type = "individual"`
**Rationale:** Business rules specify "individual" as the default for retirement accounts. Extended to all account types for consistency. Operator override is supported at approve time.

## DL-008: Business rules before Approved transition
**Date:** 2026-03-11
**Decision:** Run session validation, eligibility, deposit limit, and duplicate checks while transfer is in Analyzing state, before transitioning to Approved.
**Rationale:** Previously, business rules ran after transitioning to Approved, causing invalid state transitions (Approved→Rejected) when over-limit. Fix ensures Analyzing→Rejected for rule failures.

## DL-009: Operator approve enforces deposit limit
**Date:** 2026-03-11
**Decision:** Operator approve endpoint validates deposit limit before posting ledger.
**Rationale:** Flagged transfers bypass deposit-flow business rules; an over-limit flagged deposit could be approved. Config.CheckLimit added; operator approve rejects with 422 when over limit.

## DL-010: EOD cutoff in settlement response
**Date:** 2026-03-11
**Decision:** Settlement trigger response includes `after_eod_cutoff` boolean; settlement still proceeds.
**Rationale:** Enables monitoring and observability without blocking. Deposits after 6:30 PM CT roll to next business day per requirements; explicit flag supports operator awareness.

## DL-011: Operator authentication (cookie sessions)
**Date:** 2026-03-12
**Decision:** Operator login uses cookie-based sessions (gorilla/sessions). Passwords stored as bcrypt hashes in `operators` table. Protected routes use `auth.RequireOperator` middleware.
**Rationale:** Simple, stateless-from-server perspective, no JWT/OAuth for demo. Session secret via `SESSION_SECRET`; 7-day expiry, HttpOnly, SameSite Lax.
**Trade-off:** Production would consider CSRF tokens and secure cookie settings (Secure in prod).

## DL-012: Guest login for operator portal
**Date:** 2026-03-12
**Decision:** `POST /operator/guest` creates an ephemeral session with a generated operator ID (e.g. `guest-xxxxxxxx`). No DB record; used for demo without seeded accounts.
**Rationale:** Allows trying the operator UI (queue, approve/reject, settlement) without configuring operator accounts. `/operator/me` returns synthetic guest identity.

## DL-013: Travel clock for testing (superseded)
**Date:** 2026-03-12
**Decision:** `internal/clock.TravelClock` provided app-level time: set arbitrary time, freeze, resume. Exposed at `GET/POST /operator/clock` for test-only time travel.
**Superseded (2026-03-15):** Time travel removed. Operator UI keeps the clock display (local time only). Run `make seed-deposits` to insert 25 deposits with created_at before/after 6:30 PM CT for settlement demo.
**Rationale (original):** EOD cutoff (6:30 PM CT) and settlement batching are time-dependent; tests and demos need reproducible “business day” behavior without waiting or mocking every call site.

## DL-014: Content negotiation for operator SPA
**Date:** 2026-03-12
**Decision:** `GET /deposits` and `GET /deposits/{id}` inspect `Accept`: if it prefers `text/html`, serve the operator SPA (index.html); otherwise return JSON. Same path serves both API clients and browser navigation.
**Rationale:** Single entry point for operator UI and API; no separate `/app` or `/api/deposits` prefix for the list/detail views. Browser requests (navigation, fetch with default Accept) get HTML; API clients sending `Accept: application/json` get JSON.

## DL-015: Idempotency key optional for deposits
**Date:** 2026-03-12
**Decision:** If `X-Idempotency-Key` is omitted on `POST /deposits`, the server generates a new UUID and uses it as the idempotency key for that request. Response is still cached and replayed for duplicate keys.
**Rationale:** Clients that do not send a key still get exactly-once semantics per request (each request gets a new key). Clients that send a key get replay of the same response on retries.
**Trade-off:** Without a client-supplied key, retries are not deduplicated across duplicate submissions; acceptable for MVP.

## DL-016: Per-deposit decision trace (observability)
**Date:** 2026-03-15
**Decision:** Structured JSON log lines prefixed with `DEPOSIT_TRACE` for each deposit at key stages: vendor_response, vendor_flagged, business_rules (with rule if rejected), funds_posted, operator_action (approve/reject), settlement_status. No PII; only transfer_id, account_id (synthetic), and stage-specific fields.
**Rationale:** Meets requirement for "per-deposit decision trace: inputs → Vendor response → business rules → operator actions → settlement status" and supports debugging without a separate trace store for MVP.
