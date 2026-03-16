# Grade: Mobile Check Deposit App vs. requirements.md

Grading against the Evaluation Rubric (100 pts total) and Must-Have requirements in `docs/requirements.md`.

---

## Evaluation Rubric

### 1. System design and architecture (20 pts) — **18/20**

**Strengths:**
- **Service boundaries:** Clear separation in `internal/`: `transfer`, `vendor`, `funding`, `ledger`, `operator`, `settlement`, `return_`, `auth`, `clock`, `api`. Architecture doc describes HTTP → orchestration → repos/services.
- **State machine:** Documented in `docs/architecture.md` with valid transitions table; implemented in `internal/transfer/state.go` with `validTransitions` and `Transfer.Transition()`.
- **Data flow:** Requested → Validating → Analyzing → Approved → FundsPosted → Completed (happy path) and Rejected/Returned (non-happy) are correct.
- **Decision log:** `docs/decision_log.md` documents 16+ decisions (SQLite, in-process stub, net/http, X9-like JSON, return fee, session/guest, idempotency, DEPOSIT_TRACE, etc.) with rationale and trade-offs.

**Deductions:**
- **-2:** `docs/architecture.md` is strong but could include a single end-to-end sequence diagram (investor → vendor → funding → ledger → settlement) for even clearer flow. Minor.

---

### 2. Core correctness (25 pts) — **24/25**

**Strengths:**
- **Happy path:** POST /deposits → job queue → ProcessDeposit: Validating → vendor stub (clean_pass) → business rules → Approved → ledger (CreateMovementEntry with To=investor, From=omnibus) → FundsPosted. Confirmed in `internal/api/deposits.go` and tests.
- **Business rules:** Session/eligibility (`CheckEligibility` via omnibus lookup), deposit limit ($5,000 in `funding.Config`), duplicate (transaction_id), contribution default "individual"; operator approve also enforces limit (DL-009).
- **State transitions:** `state_test.go` covers valid and invalid transitions; deposit and operator tests exercise Requested→Validating→Analyzing→Approved→FundsPosted and Rejected/Returned.
- **Ledger postings:** `ledger.go` uses required attributes: Type=MOVEMENT, Memo=FREE, SubType=DEPOSIT, TransferType=CHECK, Currency=USD, SourceApplicationID=transferID; ToAccountID=investor, FromAccountID=omnibus from `funding.GetOmnibusAccount`.

**Deductions:**
- **-1:** Deposit submission is async (202 + job queue); spec says "ledger posting created" in the flow—tests process jobs synchronously via ProcessOneJob/ProcessOneJobHTTP, so behavior is correct but the main path is async. No functional error; small clarity gap in “immediate” posting wording.

---

### 3. Vendor Service stub quality (15 pts) — **15/15**

**Strengths:**
- **Differentiated scenarios:** All seven required scenarios implemented in `internal/vendor/stub.go`: IQA pass, IQA fail (blur), IQA fail (glare), MICR read failure, duplicate, amount mismatch, clean pass.
- **Configurable without code changes:** `config/scenarios.json` maps account ID (e.g. `ACC-IQA-BLUR`) to scenario; stub uses `resolveScenario(req, scenarioOverride)` with prefix fallback and `default_scenario`.
- **Deterministic:** Same account ID always yields same scenario; scenario override supported for tests. Documented in README “Test Accounts” table.
- **Tests:** `vendor/stub_test.go` and `api/scenarios_test.go` cover each scenario; `api/deposits_test.go` covers IQA blur/glare, over-limit, MICR fail, idempotency, etc.

---

### 4. Operator workflow and observability (10 pts) — **9/10**

**Strengths:**
- **Review queue:** GET /operator/queue returns flagged (Analyzing) transfers with IQScore, MICRConfidence, investor display name; transfer includes vendor_response, front_image_path, back_image_path, micr_data, ocr_amount, entered_amount (from DB). Filter by date, account, amount_min, amount_max.
- **Approve/reject with logging:** POST /operator/approve and /operator/reject; `operator_actions` table stores action, operator_id, note, contribution_type_override, created_at. GET /operator/audit and GET /operator/actions/{id} for audit trail.
- **Contribution override:** ApproveRequest accepts `contribution_type` override.
- **Decision trace:** DEPOSIT_TRACE log lines for vendor_response, vendor_flagged, business_rules, funds_posted, operator_action, settlement_status (see decision_log DL-016). Redacted (no real PII).
- **Settlement monitoring:** GET /health/settlement returns unsettled count and EOD cutoff state.

**Deductions:**
- **-1:** Operator endpoints require login (cookie or guest). CLI demo script (`scripts/demo.sh`) does not call POST /operator/guest or /operator/login before approve/reject, so those steps return 401 when run standalone. README recommends Playwright demo for full flow; fixing the script would make CLI demo self-contained.

---

### 5. Return/reversal handling (10 pts) — **10/10**

**Strengths:**
- **Return processing:** POST /returns with transfer_id, reason, optional fee. `return_/reversal.go` validates state (FundsPosted or Completed), creates reversal entry (debit investor, credit omnibus), applies $30 fee (ReturnFee constant), transitions to Returned.
- **Reversal entry:** CreateReversalEntry with amount and fee; GetAccountBalance accounts for reversal_fee in debits.
- **State transition:** Only FundsPosted and Completed can be returned; transfer moves to Returned; settlement metadata cleared when returning from FundsPosted.
- **Investor notification:** Log line "RETURN NOTIFICATION" (stub per risks doc L-005).
- **Tests:** `return_/reversal_test.go` covers FundsPosted, Completed, wrong state, not found, custom fee, and exclusion from settlement after return.

---

### 6. Tests and evaluation rigor (10 pts) — **10/10**

**Strengths:**
- **Count:** Well over 10 test functions (50+ across `api/deposits_test.go`, `api/operator_test.go`, `api/scenarios_test.go`, `vendor/stub_test.go`, `funding/funding_test.go`, `transfer/state_test.go`, `ledger/ledger_test.go`, `return_/reversal_test.go`, `settlement/engine_test.go`, etc.).
- **Coverage:** Happy path, each vendor scenario (IQA blur/glare, MICR fail, duplicate, amount mismatch, over-limit), business rules (limit, duplicate, eligibility, contribution default), state machine (valid/invalid), operator approve/reject and over-limit on approve, reversal with fee, settlement file generation and EOD rollover.
- **Test report:** `make test-report` produces `reports/test_report.txt` with verbose test output (deliverable).

---

### 7. Developer experience (10 pts) — **9/10**

**Strengths:**
- **One-command setup:** `make dev` runs the server; `docker-compose up` available. README has clear Prerequisites, Install & Run, Run Tests, Build, Docker.
- **README:** Architecture overview (ASCII flow), transfer states, setup, test accounts table, how to demo (Playwright + CLI), API endpoints, env vars. Concise and usable.
- **Demo scripts:** `scripts/demo.sh` runs 11 scenarios (clean pass, IQA blur/glare, MICR fail, operator approve, amount mismatch reject, over-limit, duplicate, retirement, return, idempotency, settlement, ledger, balance). E2E Playwright demo (`make demo` / `make demo-headed`) for full UI flow.
- **Decision log:** Present and detailed.
- **.env.example:** Present with DATABASE_URL and PORT. Session secret (SESSION_SECRET) is optional with default in code; could be listed in .env.example for production clarity.

**Deductions:**
- **-1:** CLI demo does not authenticate for operator steps (see Operator workflow above). Small fix: add `POST /operator/guest` (and set cookie) before operator approve/reject in `scripts/demo.sh`.

---

## Rubric total: **95/100**

---

## Functional Requirements (Must-Haves) — Checklist

| Requirement | Status | Notes |
|-------------|--------|--------|
| Endpoint/UI for mobile check deposit simulation (front/back image, amount, account) | Met | POST /deposits; mobile UI at /mobile; sandbox at /sandbox |
| Vendor stub accepts image payloads, returns structured validation results | Met | POST /vendor/validate; stub returns status, reason, scores, etc. |
| Re-submission on IQA failure with actionable messages | Met | 422 with reason (blur, glare); client can retry with new images |
| IQA Pass | Met | clean_pass / iqapass |
| IQA Fail (Blur) | Met | iqafail_blur |
| IQA Fail (Glare) | Met | iqafail_glare |
| MICR Read Failure | Met | micr_fail → flagged |
| Duplicate Detected | Met | duplicate → reject |
| Amount Mismatch | Met | amount_mismatch → flagged |
| Clean Pass with MICR/amounts/transaction ID | Met | clean_pass returns full data |
| Stub configurable/deterministic (account or config) | Met | config/scenarios.json + account prefix |
| Funding: validate session & eligibility | Met | ValidateSession, CheckEligibility (omnibus lookup) |
| Resolve account to internal/omnibus | Met | funding.Config.GetOmnibusAccount |
| Business rules: limit $5k, contribution default, duplicate | Met | CheckLimit, GetContributionDefault, CheckDuplicate |
| Ledger attributes (To/From, Type, Memo, SubType, TransferType, Currency, Amount, SourceApplicationId) | Met | ledger.CreateMovementEntry |
| Transfer states: Requested→Validating→Analyzing→Approved→FundsPosted→Completed; Rejected; Returned | Met | state.go + transitions |
| Operator queue: images, MICR, risk, amount comparison | Met | QueueItem + Transfer fields; filter/search |
| Approve/reject with action logging | Met | operator_actions + audit endpoints |
| Override contribution type | Met | ApproveRequest.contribution_type |
| Search/filter by date, status, account, amount | Met | ListFlaggedTransfers params |
| Audit log who/what/when | Met | operator_actions, GET /operator/audit |
| Settlement file: MICR, image refs, amount, batch metadata | Met | SettlementEntry in JSON; EOD cutoff |
| EOD cutoff 6:30 PM CT; late rolls to next business day | Met | settlement/eod.go; seed-deposits for demo |
| Return: reversal, $30 fee, Returned state, notify investor | Met | return_/reversal.go; log notification |
| Per-deposit decision trace; redacted logs | Met | DEPOSIT_TRACE; risks doc |
| Synthetic data only; secrets via env; .env.example | Met | .env.example; no real PII |
| One-command setup | Met | make dev; docker-compose up |
| Risks/limitations note | Met | docs/risks_limitations.md |

---

## Deliverables Checklist

| Deliverable | Status |
|-------------|--------|
| README — setup, architecture, flow, demo, disclaimers | Yes |
| docs/decision_log.md | Yes |
| docs/architecture.md | Yes |
| /tests (unit + integration, min 10) | Yes (tests under internal/*_test.go) |
| /reports (test results, scenario coverage) | Yes (reports/test_report.txt) |
| .env.example | Yes |
| Vendor stub documented (scenarios, config) | Yes (README table + config/scenarios.json) |
| Demo scripts (happy, rejection, review, return) | Yes (scripts/demo.sh + e2e Playwright) |
| Short write-up: architecture, stub, state machine, risks | Yes (architecture.md, decision_log, risks_limitations) |

---

## Summary

- **Grade: 95/100.** The app meets the spec’s success criteria and almost all must-haves. Strong areas: state machine, vendor stub, ledger and reversal logic, test count and coverage, operator workflow and audit, return/reversal, and documentation.
- **Improvements that would raise the score:**
  1. Add operator auth (e.g. guest login + cookie) to `scripts/demo.sh` so operator approve/reject succeed in the CLI demo.
  2. Optionally add SESSION_SECRET to .env.example and a one-page architecture/stub/state-machine summary if a separate short write-up is expected beyond the existing docs.
