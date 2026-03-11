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
