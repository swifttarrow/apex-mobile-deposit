# User Stories

Verifiable scenarios for the Checkstream mobile check deposit pipeline. These stories map to PRD requirements and testing scenarios and can be exercised via API, UI, or test fixtures.

---

## Conventions

- **As a** role **I want** capability **so that** outcome
- **Verification:** How to confirm the story passes
- Stories are ordered by flow priority (deposit capture → funding → ledger → operator → settlement → returns)

---

## 1. Deposit Capture & Vendor Integration

### US-1.1: Happy path — clean pass to Completed

**As a** user **I want** to submit a check with valid images and amount **so that** the deposit is processed end-to-end without operator review.

**Verification:**

- Submit deposit (front image, back image, amount, account) with account configured for clean pass (e.g. `ACC-001` or `X-Test-Scenario: clean_pass`; see `config/scenarios.json`)
- Transfer reaches `Completed`
- Ledger has MOVEMENT entry: To=investor, From=omnibus, SubType=DEPOSIT, TransferType=CHECK, Memo=FREE, Currency=USD
- Settlement file includes deposit with MICR data, images, amount
- No operator action required

---

### US-1.2: IQA fail (blur) — retake prompt

**As a** user **I want** actionable feedback when my check image is blurry **so that** I know to retake the photo.

**Verification:**

- Submit deposit with account `ACC-IQA-BLUR` (or equivalent stub trigger)
- Response returns error with reason `blur` and message indicating retake (e.g. "Image too blurry")
- Transfer reaches `Rejected` (from Validating)
- No ledger posting
- No settlement file entry

---

### US-1.3: IQA fail (glare) — retake prompt

**As a** user **I want** actionable feedback when glare is detected **so that** I know to retake in better lighting.

**Verification:**

- Submit deposit with account `ACC-IQA-GLARE` (or equivalent stub trigger)
- Response returns error with reason `glare` and message indicating retake (e.g. "Glare detected")
- Transfer reaches `Rejected` (from Validating)
- No ledger posting
- No settlement file entry

---

### US-1.4: MICR read failure — flagged for operator review

**As a** system **I want** deposits with MICR read failure to be flagged for operator review **so that** a human can approve or reject.

**Verification:**

- Submit deposit with account `ACC-MICR-FAIL`
- Transfer reaches `Analyzing` (flagged)
- Deposit appears in operator review queue with check images and MICR data (or "unreadable")
- Operator can approve → ledger posted, transfer → Approved → FundsPosted → Completed
- Operator can reject → transfer → Rejected, no ledger posting

---

### US-1.5: Duplicate detected — rejection

**As a** system **I want** duplicate check deposits to be rejected **so that** funds are not double-posted.

**Verification:**

- Submit deposit with account `ACC-DUP-001` (or second submission of same check)
- Transfer reaches `Rejected` (from Validating)
- Response indicates duplicate
- No ledger posting
- No settlement file entry

---

### US-1.6: Amount mismatch — flagged for operator review

**As a** system **I want** deposits with amount mismatch (OCR vs entered) to be flagged **so that** an operator can approve or reject.

**Verification:**

- Submit deposit with account/params triggering `amount_mismatch` (e.g. entered $1500, OCR $150)
- Transfer reaches `Analyzing` (flagged)
- Operator queue shows both entered and OCR amounts
- Operator approves → ledger posts entered amount (or policy-defined amount), flow continues
- Operator rejects → transfer → Rejected, no ledger posting

---

### US-1.7: Stub configurability without code changes

**As a** tester/evaluator **I want** to trigger different vendor scenarios via config or request parameters **so that** I can exercise all paths without code changes.

**Verification:**

- At least 6 distinct stub responses achievable via: test account prefix, header (`X-Test-Scenario`), or `config/scenarios.json`
- Scenarios in default config: clean pass, IQA fail blur, IQA fail glare, MICR fail, duplicate, amount mismatch (see `config/scenarios.json` and vendor stub)
- Changing config or request produces expected stub response; no redeploy required

---

## 2. Funding Service & Business Rules

### US-2.1: Over-limit deposit rejected

**As a** system **I want** deposits over the $5,000 MVP limit to be rejected **so that** business rules are enforced.

**Verification:**

- Submit clean-pass deposit with amount > $5,000
- Transfer reaches `Rejected` (from Analyzing)
- No ledger posting
- Response indicates limit exceeded

---

### US-2.2: Under-limit deposit accepted

**As a** user **I want** deposits within the limit to proceed **so that** valid deposits are not blocked.

**Verification:**

- Submit clean-pass deposit with amount ≤ $5,000
- Deposit proceeds through happy path
- Ledger and settlement include deposit

---

### US-2.3: Duplicate detection by Funding Service

**As a** system **I want** Funding Service to enforce duplicate detection **so that** no duplicate deposits are posted even if Vendor passes.

**Verification:**

- Two deposits with same check identifier (or duplicate-inducing account)
- First succeeds
- Second is rejected by Funding Service; transfer → Rejected, no ledger posting

---

### US-2.4: Account and session validation

**As a** system **I want** account identifiers resolved to internal accounts **so that** ledger posting uses correct To/From accounts.

**Verification:**

- Submit deposit with valid account; ledger entry has correct To AccountId (investor) and From AccountId (omnibus)
- Invalid or unknown account returns appropriate error; no ledger posting

---

## 3. Ledger & State Machine

### US-3.1: Correct MOVEMENT entry on approval

**As a** system **I want** ledger posting to create correct MOVEMENT entries **so that** balances and audit trail are accurate.

**Verification:**

- After approval, ledger entry has:
  - Type: MOVEMENT
  - To AccountId: investor
  - From AccountId: omnibus
  - SubType: DEPOSIT
  - Transfer Type: CHECK
  - Memo: FREE
  - Currency: USD
  - Amount: deposit amount

---

### US-3.2: Valid state transitions only

**As a** system **I want** invalid state transitions to be rejected **so that** transfer lifecycle remains consistent.

**Verification:**

- Requested → Validating (on submit)
- Validating → Rejected (IQA fail, duplicate)
- Validating → Analyzing (IQA pass/flagged)
- Analyzing → Rejected (operator reject, business rule fail)
- Analyzing → Approved (auto or operator approve)
- Approved → FundsPosted (ledger posted)
- FundsPosted → Completed (settlement)
- FundsPosted → Returned (return received)
- Invalid transition attempts (e.g. Rejected → Approved) are rejected

---

### US-3.3: State queryable within 1 second

**As a** operator or client **I want** transfer state to be queryable within 1 second of a transition **so that** UI and workflows stay in sync.

**Verification:**

- After deposit submission, state change (e.g. Validating → Analyzing) is visible via `GET /deposits/{id}` within 1 second
- Performance target: queryable within 1 second

---

## 4. Operator Workflow

**Note:** Operator and settlement endpoints require authentication. Use `POST /operator/login` (e.g. username/password for seeded operators) or `POST /operator/guest`; send the session cookie (or same browser session) for `GET /operator/queue`, `POST /operator/approve`, `POST /operator/reject`, `GET /operator/audit`, `POST /settlement/trigger`, etc. See [architecture](architecture.md#api-routes-summary) and [DL-011](decision_log.md#dl-011-operator-authentication-cookie-sessions).

### US-4.1: Review queue displays flagged deposits

**As a** operator **I want** to see all flagged deposits with images, MICR data, and amounts **so that** I can make approve/reject decisions.

**Verification:**

- (When logged in) Operator queue lists deposits in `Analyzing` with flags (MICR fail, amount mismatch)
- Each row shows: check images (front/back), MICR data, entered amount, OCR amount (if mismatch), risk context
- Search/filter by date, account, amount works (query params: `date`, `account`, `amount_min`, `amount_max`, `limit`, `offset`)

---

### US-4.2: Approve/reject actions logged

**As a** auditor **I want** operator actions to be logged with who, what, when **so that** there is a full audit trail.

**Verification:**

- Operator approve or reject creates `operator_actions` record
- Record includes: operator identity, action (approve/reject), timestamp, transfer ID
- Logs are queryable for compliance

---

### US-4.3: Flagged items visible within 1 second

**As a** operator **I want** newly flagged deposits to appear in the queue within 1 second **so that** I can respond promptly.

**Verification:**

- Deposit flagged (e.g. MICR fail) appears in operator queue within 1 second of being flagged
- Performance target: visible within 1 second

---

## 5. Settlement

### US-5.1: Settlement file generated with required data

**As a** settlement processor **I want** a settlement file (X9 ICL or equivalent) with MICR data, images, and amounts **so that** the bank can process deposits.

**Verification:**

- EOD settlement run produces file (X9 ICL or structured JSON)
- File includes: MICR data, check images, amounts, batch metadata
- File generation completes within 5 seconds of EOD trigger

---

### US-5.2: EOD cutoff — deposits after 6:30 PM CT roll to next business day

**As a** system **I want** deposits submitted after 6:30 PM CT to roll to the next business day **so that** EOD cutoff is enforced.

**Verification:**

- Deposit submitted before 6:30 PM CT → included in same-day settlement batch
- Deposit submitted after 6:30 PM CT → included in next business day batch
- Cutoff is enforced at 6:30 PM CT (timezone-aware)

---

### US-5.3: Settlement file generation latency

**As a** operations team **I want** settlement file generation to complete within 5 seconds **so that** EOD processes run on time.

**Verification:**

- Trigger settlement file generation
- File is produced within 5 seconds
- Performance target: < 5 seconds from EOD trigger

---

### US-5.4: Trigger settlement includes only transfers eligible for trigger business day

**As a** settlement processor **I want** settlement trigger to include only deposits eligible for the trigger business day **so that** after-cutoff deposits are not settled early.

**Verification:**

- Create two `FundsPosted` deposits on the same CT day:
  - Deposit A before 6:30 PM CT
  - Deposit B after 6:30 PM CT
- Trigger settlement for that CT business day
- Deposit A is included and transitions `FundsPosted -> Completed`
- Deposit B is excluded and remains `FundsPosted` for next business day settlement

---

### US-5.5: Settlement trigger transitions only included transfers to Completed

**As a** system **I want** only transfers actually written to the settlement file to transition to `Completed` **so that** transfer state reflects true settlement inclusion.

**Verification:**

- Trigger settlement when both eligible and deferred `FundsPosted` deposits exist
- Every transfer included in the generated settlement file has:
  - `state = Completed`
  - settlement metadata populated (`settlement_batch_id`, `settlement_ack_at`)
- Every deferred transfer not in the file remains `FundsPosted`

---

## 6. Return & Reversal

### US-6.1: Return triggers reversal and fee

**As a** system **I want** a bounced check to trigger reversal posting with a $30 fee **so that** funds and fees are correctly accounted for.

**Verification:**

- Simulate return notification for a Completed (or FundsPosted) deposit
- Reversal posting: debit investor for original amount + $30 fee
- Transfer transitions to `Returned`
- Ledger has reversal entries (credit omnibus, debit investor, fee)

---

### US-6.2: Return after FundsPosted

**As a** system **I want** returns that occur before settlement to be handled **so that** the transfer moves to Returned and no settlement file entry is sent.

**Verification:**

- Deposit in `FundsPosted` (not yet settled)
- Return notification received
- Reversal posted, transfer → Returned
- Deposit excluded from (or removed from) settlement batch if not yet sent

---

### US-6.3: Return after Completed

**As a** system **I want** returns that occur after settlement to be handled **so that** reversal is posted and investor is debited.

**Verification:**

- Deposit in `Completed`
- Return notification received (e.g. `POST /returns` with `transfer_id`, `reason`)
- Reversal posted (amount + $30 fee), transfer → Returned
- Ledger reflects reversal

**Implementation note:** Return service accepts both `FundsPosted` and `Completed`, but the state machine currently allows only `FundsPosted → Returned`. If return from `Completed` fails with invalid transition, see [L-005](risks_limitations.md#l-005-return-from-completed-not-implemented).

---

### US-6.4: Return before settlement clears pending settlement association

**As a** system **I want** a return on a `FundsPosted` (not yet settled) deposit to clear pending settlement association **so that** the deposit is excluded from settlement and cannot be accidentally settled later.

**Verification:**

- Deposit is in `FundsPosted` and has not been settled
- Return notification is processed
- Transfer transitions to `Returned`
- Transfer is not included in subsequent settlement file generation
- Settlement metadata is empty for the returned transfer (`settlement_batch_id`, `settlement_ack_at`)

---

## 7. Setup & Deployment

### US-7.1: One-command setup

**As a** developer **I want** to run one command to start the system **so that** I can develop and test locally.

**Verification:**

- `make dev` (Go server) or `docker compose up` (if using Docker) starts the system
- README documents setup steps (e.g. CGO for SQLite, `config/scenarios.json`)
- System accepts deposits and processes them

---

### US-7.2: Publicly accessible deployment

**As a** evaluator **I want** the application to be deployed and publicly accessible **so that** I can verify functionality without local setup.

**Verification:**

- Application deployed (Railway, Render, Fly.io, etc.)
- Public URL returns healthy response
- Deposit submission works via public endpoint

---

## 8. Performance Targets (Summary)


| Scenario                     | Target                                |
| ---------------------------- | ------------------------------------- |
| Vendor stub response         | < 1 second                            |
| Ledger posting latency       | < 5 seconds from approval             |
| Settlement file generation   | < 5 seconds from EOD trigger          |
| Operator queue update        | Flagged items visible within 1 second |
| State transition propagation | Queryable within 1 second             |


---

## 9. PRD Testing Scenario Checklist


| #   | PRD Scenario                                                                      | User Story             |
| --- | --------------------------------------------------------------------------------- | ---------------------- |
| 1   | Happy path: submit → validate → approve → ledger → settlement → Completed         | US-1.1                 |
| 2   | IQA fail (blur): retake prompt, no ledger                                         | US-1.2                 |
| 3   | IQA fail (glare): same as blur                                                    | US-1.3                 |
| 4   | MICR read failure: flagged, operator approves → ledger posted                     | US-1.4                 |
| 5   | Duplicate detected: rejected, no ledger                                           | US-1.5, US-2.3         |
| 6   | Amount mismatch: flagged, operator approves or rejects                            | US-1.6                 |
| 7   | Over-limit deposit (>$5,000): rejected by Funding                                 | US-2.1                 |
| 8   | Return/reversal: reversal + fee, transfer Returned                                | US-6.1, US-6.2, US-6.3 |
| 9   | EOD cutoff: after 6:30 PM CT rolls to next business day                           | US-5.2                 |
| 10  | Stub configurability: different inputs → different responses without code changes | US-1.7                 |
| 11  | Settlement trigger: only eligible transfers included/completed; deferred stay pending | US-5.4, US-5.5      |
| 12  | Pre-settlement return: returned transfer excluded from settlement batch              | US-6.4               |


---

## 10. Idempotency (Interview Topic)

**As a** system **I want** deposit submission and return processing to be idempotent **so that** duplicate requests do not double-post.

**Verification (non-blocking for MVP):**

- Same `X-Idempotency-Key` on two deposit submissions → second returns cached response, no second transfer
- Same return notification processed twice → reversal posted once, no double fee

