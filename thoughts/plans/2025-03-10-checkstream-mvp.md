# Checkstream MVP Implementation Plan

## Overview

Build a monolithic Go mobile check deposit pipeline (Checkstream) with SQLite: deposit capture, Vendor stub validation, Funding Service, transfer state machine, operator review, settlement file (X9 ICL), and return/reversal handling. Optimized for speed; one-command setup; deployed and publicly accessible.

---

## Current State Analysis

- **Codebase:** Greenfield. No Go code, no existing services.
- **Docs:** PRD (`docs/prd.md`), requirements (`docs/requirements.md`), terminology (`docs/terminology.md`).
- **Research:** Pre-Search complete (`thoughts/research/checkstream-prd-research.md`). Locked: Go, SQLite, in-process stub, scenarios.json, Moov X9 ICL.
- **Constraints:** MVP gate in 24 hrs; optimize for speed; monolithic architecture.

---

## Desired End State

All MVP requirements from PRD satisfied:

- Deposit submission endpoint/UI accepts images (front/back), amount, account
- Vendor stub returns 7+ differentiated scenarios (config/params)
- Funding enforces $5K limit, contribution defaults, duplicate detection
- Transfer state machine: 8 states, valid transitions only
- Ledger posting: MOVEMENT entries (To/From, SubType DEPOSIT, Transfer Type CHECK)
- Operator queue: flagged deposits, images, MICR, approve/reject, audit log
- Settlement file: X9 ICL with MICR, images, amounts; EOD cutoff 6:30 PM CT
- Return/reversal: $30 fee, reversal posting, state → Returned
- One-command setup (`make dev` or `docker compose up`)
- Deployed and publicly accessible

**Verification:** Run all 10 testing scenarios; `make test` passes; demo script exercises all paths.

---

## What We're NOT Doing

- Real Vendor API integration (stub only)
- Real Settlement Bank (file to disk; optional stub ack)
- Operator authentication (defer; document limitation)
- Production observability (metrics, traces)
- S3/blob storage for images (file path or base64 in DB)
- Elasticsearch for search (simple SQL WHERE)
- Idempotency for MVP if time-constrained (document as follow-up)

---

## Phase 1: Project Setup & Transfer State Machine

### Overview

Initialize Go module, directory layout, SQLite schema, and transfer state machine with valid transitions. Foundation for all subsequent phases.

### Changes Required

**New files:**

| Path | Purpose |
|------|---------|
| `go.mod` | Go module (e.g. `github.com/.../checkstream`) |
| `Makefile` | `make dev`, `make test`, `make build` |
| `cmd/server/main.go` | Entry point; HTTP server stub |
| `internal/db/schema.sql` | SQLite DDL: transfers, ledger_entries, operator_actions, idempotency_keys, check_images |
| `internal/db/db.go` | SQLite connection, migrations |
| `internal/transfer/state.go` | State enum, valid transitions map |
| `internal/transfer/transfer.go` | Transfer model, transition validation |
| `internal/transfer/repository.go` | CRUD, state updates |
| `config/scenarios.json` | Stub scenario mapping (placeholder) |

**Schema (transfers):**

- `id`, `account_id`, `amount`, `state`, `vendor_response` (JSON), `created_at`, `updated_at`
- `front_image_path`, `back_image_path` (or base64 columns)
- `micr_data` (JSON), `ocr_amount`, `entered_amount`, `transaction_id`

**Valid transitions (from PRD):**

- Requested → Validating
- Validating → Rejected | Analyzing
- Analyzing → Rejected | Approved
- Approved → FundsPosted
- FundsPosted → Completed | Returned

### Success Criteria

#### Automated Verification

- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/transfer/...` — state transition validation tests pass
- [ ] `make dev` starts server (or fails gracefully if DB not ready)

#### Manual Verification

- [ ] Schema applied; `transfers` table exists
- [ ] Invalid transition (e.g. Rejected → Approved) is rejected in code

**Note:** Pause for human confirmation after this phase before proceeding.

---

## Phase 2: Vendor Service Stub

### Overview

Config-driven stub returning 7+ differentiated responses. Selection via `scenarios.json` (account prefix) and `X-Test-Scenario` header.

### Changes Required

**Files:**

| Path | Changes |
|------|---------|
| `config/scenarios.json` | Map account prefixes → scenario names (e.g. `ACC-IQA-BLUR` → `iqafail_blur`) |
| `internal/vendor/stub.go` | Stub handler: load config, resolve scenario from account/header, return JSON |
| `internal/vendor/types.go` | Request/response structs (IQA, MICR, OCR, duplicate, amount_mismatch, clean_pass) |
| `cmd/server/main.go` | Register `POST /vendor/validate` route |

**Scenarios to implement:**

1. `iqafail_blur` — `{"status":"fail","reason":"blur","message":"Image too blurry"}`
2. `iqafail_glare` — `{"status":"fail","reason":"glare","message":"Glare detected"}`
3. `micr_fail` — `{"status":"flagged","reason":"micr_fail"}`
4. `duplicate` — `{"status":"reject","reason":"duplicate"}`
5. `amount_mismatch` — `{"status":"flagged","reason":"amount_mismatch","ocrAmount":150,"enteredAmount":1500}`
6. `clean_pass` — `{"status":"pass","micr":{...},"amount":150,"transactionId":"..."}`
7. `iqapass` — `{"status":"pass","iqScore":0.95}` (proceed to MICR/OCR)

### Success Criteria

#### Automated Verification

- [ ] `go test ./internal/vendor/...` — each scenario returns expected JSON
- [ ] Different account IDs produce different responses per config

#### Manual Verification

- [ ] `curl -X POST .../vendor/validate` with `account_id: ACC-IQA-BLUR` returns blur fail
- [ ] `X-Test-Scenario: clean_pass` overrides account-based selection

**Note:** Pause for human confirmation after this phase before proceeding.

---

## Phase 3: Funding Service & Ledger

### Overview

Business rules ($5K limit, duplicate detection), omnibus lookup, ledger posting. MOVEMENT entries with To/From, SubType DEPOSIT, Transfer Type CHECK.

### Changes Required

**Files:**

| Path | Changes |
|------|---------|
| `internal/ledger/ledger.go` | Ledger posting: create MOVEMENT entry |
| `internal/ledger/schema.sql` | `ledger_entries` table (or extend `internal/db/schema.sql`) |
| `internal/funding/funding.go` | Business rules: limit check, duplicate check, account resolution |
| `internal/funding/config.go` | Omnibus map (account_id → omnibus_id); $5K limit |
| `internal/db/schema.sql` | Add ledger_entries if not in Phase 1 |

**Ledger entry fields (from PRD):**

- To AccountId (investor), From AccountId (omnibus)
- Type: MOVEMENT, Memo: FREE, SubType: DEPOSIT, Transfer Type: CHECK, Currency: USD
- Amount, SourceApplicationId (TransferID)

**Funding rules:**

- Reject if amount > $5,000
- Reject if duplicate (same check/transaction ID)
- Resolve account_id → omnibus_id via config

### Success Criteria

#### Automated Verification

- [ ] `go test ./internal/funding/...` — over-limit rejected, duplicate rejected
- [ ] `go test ./internal/ledger/...` — MOVEMENT entry created with correct fields

#### Manual Verification

- [ ] Ledger entry matches PRD spec for a valid deposit

**Note:** Pause for human confirmation after this phase before proceeding.

---

## Phase 4: Deposit Submission Flow

### Overview

REST API: `POST /deposits` accepts images, amount, account. Orchestrates Vendor stub → Funding → ledger + state updates. Idempotency via `X-Idempotency-Key`.

### Changes Required

**Files:**

| Path | Changes |
|------|---------|
| `internal/api/deposits.go` | POST /deposits handler |
| `internal/api/middleware.go` | Idempotency middleware (optional for MVP) |
| `cmd/server/main.go` | Wire: deposits handler, Vendor stub, Funding, ledger |
| `internal/transfer/repository.go` | Create transfer, update state in transaction with ledger |

**Flow:**

1. Parse request (images, amount, account_id)
2. Check idempotency key (if provided)
3. Create transfer in Requested
4. Call Vendor stub → get response
5. If fail/reject → transition to Rejected, return
6. If flagged → transition to Analyzing, return (operator must act)
7. If pass → call Funding (limits, duplicates)
8. If Funding rejects → Rejected
9. If Funding approves → Approved, post ledger, FundsPosted
10. Return transfer with state

**Endpoints:**

- `POST /deposits` — submit deposit
- `GET /deposits/:id` or `GET /transfers/:id` — status

### Success Criteria

#### Automated Verification

- [ ] `go test ./internal/api/...` — happy path, IQA fail, duplicate, over-limit
- [ ] Integration test: ACC-IQA-BLUR → Rejected, no ledger

#### Manual Verification

- [ ] Happy path: clean pass, under limit → FundsPosted, ledger entry
- [ ] IQA blur/glare → Rejected, actionable message
- [ ] Over $5K → Rejected

**Note:** Pause for human confirmation after this phase before proceeding.

---

## Phase 5: Operator Workflow

### Overview

Review queue for flagged deposits. Approve/reject with audit logging. Search/filter by date, status, account, amount.

### Changes Required

**Files:**

| Path | Changes |
|------|---------|
| `internal/api/operator.go` | GET /operator/queue, POST /operator/approve, POST /operator/reject |
| `internal/operator/repository.go` | Query flagged transfers; record operator_actions |
| `internal/db/schema.sql` | operator_actions table (transfer_id, action, operator_id, created_at) |

**Endpoints:**

- `GET /operator/queue?status=Analyzing&date=...&account=...&amount=...` — flagged deposits with images, MICR, amounts
- `POST /operator/approve` — body: `{transfer_id, operator_id}`; transition Analyzing → Approved; post ledger; log action
- `POST /operator/reject` — body: `{transfer_id, operator_id}`; transition Analyzing → Rejected; log action

**Audit:** Every approve/reject writes to `operator_actions`.

### Success Criteria

#### Automated Verification

- [ ] `go test ./internal/api/...` — MICR fail deposit flagged; operator approve → ledger posted
- [ ] Operator actions logged

#### Manual Verification

- [ ] Flagged deposit appears in queue with images, MICR, amount comparison
- [ ] Approve → FundsPosted; Reject → Rejected

**Note:** Pause for human confirmation after this phase before proceeding.

---

## Phase 6: Settlement & Return/Reversal

### Overview

Settlement file (X9 ICL via Moov) with EOD cutoff. Return handler: reversal posting, $30 fee, state → Returned.

### Changes Required

**Files:**

| Path | Changes |
|------|---------|
| `internal/settlement/engine.go` | Batch FundsPosted by EOD date; generate X9 ICL via `moov-io/imagecashletter` |
| `internal/settlement/eod.go` | EOD cutoff logic (6:30 PM CT); deposits after cutoff roll to next business day |
| `internal/api/settlement.go` | POST /settlement/trigger — manual EOD trigger |
| `internal/return/reversal.go` | Process return: debit investor, $30 fee, credit omnibus, state → Returned |
| `internal/api/returns.go` | POST /returns — body: `{transfer_id}` or `{transaction_id}` |
| `go.mod` | Add `github.com/moov-io/imagecashletter` |

**Settlement:**

- Query transfers in FundsPosted for settlement date (respect EOD cutoff)
- Build X9: File Header, Cash Letter, Bundle, Check Detail, Image View, Controls
- Write file to disk (e.g. `./settlement/YYYYMMDD.x9`)
- Transition FundsPosted → Completed

**Return:**

- Accept transfer_id or transaction_id
- Verify transfer in FundsPosted or Completed
- Create reversal ledger entries (debit investor amount + $30, credit omnibus)
- Transition → Returned
- Idempotent: duplicate return for same transfer returns 200 with existing result

### Success Criteria

#### Automated Verification

- [ ] `go test ./internal/settlement/...` — X9 file generated with correct structure
- [ ] `go test ./internal/return/...` — reversal entries, fee, state
- [ ] EOD cutoff: deposit after 6:30 PM CT excluded from same-day batch

#### Manual Verification

- [ ] Settlement file contains MICR, images, amounts
- [ ] Return → reversal posted, $30 fee, Returned

**Note:** Pause for human confirmation after this phase before proceeding.

---

## Phase 7: Demo, Tests, One-Command Setup & Deployment

### Overview

Minimal UI or CLI for demo; tests covering all 10 scenarios; `make dev`; README; deploy to Railway/Render/Fly.io.

### Changes Required

**Files:**

| Path | Changes |
|------|---------|
| `Makefile` | `make dev` (go run or docker compose up), `make test` |
| `docker-compose.yml` | Optional: single service + SQLite volume |
| `scripts/demo.sh` or `cmd/demo/main.go` | Exercise all paths: happy, IQA fail, MICR fail, duplicate, amount mismatch, over-limit, return |
| `README.md` | Setup, architecture, test account conventions, how to demo |
| `docs/architecture.md` | System diagram, data flow |
| `docs/decision_log.md` | Key decisions (stub, state machine, settlement format) |
| `.env.example` | Required env vars |
| `internal/api/*_test.go` | Integration tests for all 10 scenarios |

**Tests (minimum 10):**

1. Happy path end-to-end
2. IQA fail blur
3. IQA fail glare
4. MICR fail → operator approve → ledger
5. Duplicate rejected
6. Amount mismatch → operator approve/reject
7. Over-limit rejected
8. Return/reversal with fee
9. EOD cutoff
10. Stub configurability (different inputs → different responses)

**Deployment:**

- Railway, Render, or Fly.io
- SQLite file or attach volume
- Public URL documented in README

### Success Criteria

#### Automated Verification

- [ ] `make test` — all tests pass
- [ ] `make dev` — server starts; demo script runs

#### Manual Verification

- [ ] README instructions work for fresh clone
- [ ] Deployed app publicly accessible
- [ ] Demo script exercises all 10 scenarios

**Note:** Pause for human confirmation after this phase before proceeding.

---

## References

- PRD: `docs/prd.md`
- Requirements: `docs/requirements.md`
- Research: `thoughts/research/checkstream-prd-research.md`
- Terminology: `docs/terminology.md`
