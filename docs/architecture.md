# Checkstream Architecture

## Service Boundaries

```
┌─────────────────────────────────────────────────────┐
│                   HTTP Layer (net/http)               │
│  POST /deposits  GET /deposits/:id  /operator/...    │
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
- `repository.go` — Queue of flagged transfers, action logging

### `internal/settlement`
- `eod.go` — EOD cutoff at 6:30 PM CT
- `engine.go` — Batch FundsPosted transfers → X9-like JSON file

**Settlement format:** The implementation uses structured JSON files (not binary X9 ICL) by design. See [Decision Log DL-004](../decision_log.md#dl-004-x9-like-json-settlement-files) for rationale. Production would use moov-io/x9 or a licensed vendor SDK.

### `internal/return_`
- `reversal.go` — Reverse ledger entries, charge $30 fee, transition → Returned

### `internal/api`
- HTTP handlers wiring all services together
- Idempotency middleware (X-Idempotency-Key header)

## Database

SQLite with WAL mode. Schema in `internal/db/schema.sql`.

Tables:
- `transfers` — Core transfer records
- `ledger_entries` — Double-entry bookkeeping
- `operator_actions` — Audit log
- `idempotency_keys` — Response cache
- `check_images` — Image storage (base64)

## Configuration

`config/scenarios.json` maps account ID prefixes to vendor scenarios. The vendor stub resolves the scenario deterministically, enabling reproducible test runs without network calls.
