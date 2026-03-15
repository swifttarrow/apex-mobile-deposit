# Checkstream Architecture

## Service Boundaries

```
┌─────────────────────────────────────────────────────┐
│                   HTTP Layer (net/http)               │
│  POST/GET /deposits  /operator/*  /settlement  ...   │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│              Orchestration (api/deposits.go)          │
│  1. Create Transfer  2. Vendor Validate  3. Rules    │
│  4. Post Ledger  5. Transition State                 │
└──┬──────────┬──────────┬──────────┬─────────────────┘
   │          │          │          │
   ▼          ▼          ▼          ▼
Transfer   Vendor     Funding    Ledger
Repo       Stub       Service    Service
(SQLite)   (config)   (rules)    (SQLite)
```

## State Machine

```
Requested
    │
    ▼ (vendor call)
Validating ──── fail/reject ──► Rejected
    │
    ▼ (IQA pass)
Analyzing ───── operator reject ► Rejected
    │
    ▼ (auto or operator approve)
Approved
    │
    ▼ (ledger posted)
FundsPosted ─── return ──► Returned
    │
    ▼ (settlement)
Completed
```

### Valid Transitions

| From | To | Trigger |
|------|----|---------|
| Requested | Validating | Deposit submitted |
| Validating | Rejected | IQA/vendor fail |
| Validating | Analyzing | IQA flagged or pass → continue |
| Analyzing | Rejected | Vendor reject or operator reject |
| Analyzing | Approved | Auto-approve (clean pass) or operator approve |
| Approved | FundsPosted | Ledger entry posted |
| FundsPosted | Completed | Settlement batch |
| FundsPosted | Returned | Return processed |
| Completed | Returned | Return processed after settlement |

## Modules

### `internal/transfer`
- `state.go` — State type and `validTransitions` map
- `transfer.go` — Transfer struct with `Transition()` method
- `repository.go` — SQLite CRUD for transfers

### `internal/vendor`
- `types.go` — Request/Response types
- `stub.go` — In-process vendor simulation from `config/scenarios.json`

### `internal/funding`
- `config.go` — Omnibus map, limits, account types
- `funding.go` — `CheckLimit`, `CheckDuplicate`, `CheckEligibility`, `ValidateSession`

### `internal/ledger`
- `ledger.go` — `CreateMovementEntry`, `CreateReversalEntry`, balance query

### `internal/operator`
- `repository.go` — Queue of flagged transfers (Analyzing state), action logging, `ListFlaggedTransfers` with filters
- `accounts.go` — Operator accounts (username/password), `GetOperatorByUsername`, `GetOperatorByID`, `SeedTestOperators` (bcrypt, password "password")

### `internal/settlement`
- `eod.go` — EOD cutoff at 6:30 PM CT
- `engine.go` — Batch FundsPosted transfers → X9-like JSON file

**Settlement format:** The implementation uses structured JSON files (not binary X9 ICL) by design. See [Decision Log DL-004](../decision_log.md#dl-004-x9-like-json-settlement-files) for rationale. Production would use moov-io/x9 or a licensed vendor SDK.

### `internal/return_`
- `reversal.go` — Reverse ledger entries, charge $30 fee, transition → Returned

### `internal/auth`
- `session.go` — Cookie-based operator session (gorilla/sessions), `GetOperatorID`, `SetOperatorSession`, `ClearSession`
- `middleware.go` — `RequireOperator` wraps handlers and returns 401 if not logged in

### `internal/clock`
- `clock.go` — `TravelClock`: set/freeze/resume app time for testing EOD and settlement

### `internal/api`
- HTTP handlers wiring all services together
- `auth.go` — `AuthHandler`: login, guest, logout, me (operator session)
- Idempotency middleware (`WithIdempotency`): X-Idempotency-Key header; if omitted, server generates a key so every request is idempotent by response caching

## Database

SQLite with WAL mode. Schema in `internal/db/schema.sql`.

Tables:
- `transfers` — Core transfer records
- `ledger_entries` — Double-entry bookkeeping
- `operator_actions` — Audit log (approve/reject with operator_id, note, contribution_type_override)
- `idempotency_keys` — Response cache for POST /deposits
- `check_images` — Image storage (base64)
- `operators` — Operator accounts (username, password_hash, display_name, email) for login

## Configuration

`config/scenarios.json` maps account ID prefixes to vendor scenarios. The vendor stub resolves the scenario deterministically, enabling reproducible test runs without network calls.

## UIs and static assets

- **Operator UI** — Embedded SPA at `/`, `/review-queue`, `/settlement`, `/deposits`, `/deposits/{id}`, `/login` (from `cmd/server/web/operator`). GET `/deposits` and GET `/deposits/{id}` use content negotiation: `Accept: text/html` returns the operator SPA, otherwise JSON API.
- **Mobile UI** — Embedded at `/mobile` (from `cmd/server/web/mobile`) for check-deposit flow.
- **Sandbox** — Scenario showcase at `/sandbox` (from `cmd/server/web/scenarios`).
- **Check images** — Static assets at `/checks/{filename}` (same files as mobile `checks/`).

## API routes (summary)

| Area | Routes |
|------|--------|
| Health | `GET /health` |
| Vendor | `POST /vendor/validate` (stub) |
| Deposits | `POST /deposits` (idempotent), `GET /deposits`, `GET /deposits/{id}` (content negotiation) |
| Operator auth | `POST /operator/login`, `POST /operator/guest`, `POST /operator/logout`, `GET /operator/me` |
| Operator workflow | `GET /operator/queue`, `GET /operator/audit`, `POST /operator/approve`, `POST /operator/reject`, `GET /operator/transfer/{id}`, `GET /operator/actions/{id}` (all require login) |
| Settlement | `POST /settlement/trigger` (requires login) |
| Time (test) | `GET /operator/clock`, `POST /operator/clock` (requires login) |
| Returns | `POST /returns` |
| Ledger | `GET /ledger`, `GET /accounts/{id}/balance` |
