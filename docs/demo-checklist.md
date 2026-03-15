# Demo Video Checklist

## Pre-Demo Setup
- [ ] `go mod tidy` completed successfully
- [ ] `go test ./...` all passing
- [ ] `make build` produces binary (`bin/checkstream`)
- [ ] Server running: `make dev` (default port 8080)
- [ ] Terminal with `jq` installed for pretty output
- [ ] **Operator routes** (steps 5, 6, 11): log in first via `POST /operator/login` (e.g. `username: "joe"`, `password: "password"`) or `POST /operator/guest`; send session cookie (or same browser session) for `GET /operator/queue`, `POST /operator/approve`, `POST /operator/reject`, `POST /settlement/trigger`

## Demo Flow

### 1. Health Check
- [ ] `GET /health` returns `{"status":"ok","service":"checkdepot","version":"1.0.0"}`

### 2. Clean Pass Deposit
- [ ] POST to `/deposits` with `ACC-001`, $150
- [ ] Response: `201 Created`, state=`FundsPosted`
- [ ] Show transaction_id and MICR data in response

### 3. GET Deposit Status
- [ ] `GET /deposits/{id}` for the above transfer (path param `{id}` = transfer id from step 2)
- [ ] Show state=`FundsPosted`

### 4. IQA Failures
- [ ] POST with `ACC-IQA-BLUR` → `422 Unprocessable Entity`, reason=blur
- [ ] POST with `ACC-IQA-GLARE` → `422 Unprocessable Entity`, reason=glare

### 5. Operator Workflow (MICR Fail)
- [ ] POST with `ACC-MICR-FAIL` → `202 Accepted`, state=`Analyzing`
- [ ] (Ensure logged in) `GET /operator/queue` → shows flagged transfer
- [ ] POST `/operator/approve` with `transfer_id` → state=`FundsPosted`

### 6. Amount Mismatch → Operator Rejects
- [ ] POST with `ACC-MISMATCH` → `202 Accepted`, state=`Analyzing`
- [ ] (Ensure logged in) POST `/operator/reject` with `transfer_id` → state=`Rejected`

### 7. Over Deposit Limit
- [ ] POST with `ACC-OVER-LIMIT`, $6000 → `422`, over limit error

### 8. Retirement Account
- [ ] POST with `ACC-RETIRE-001`, $1000
- [ ] Response: `201`, `contribution_type=individual`

### 9. Return/Reversal
- [ ] POST deposit with `ACC-001`, $500 → `201 FundsPosted`; note the `id` in the response
- [ ] POST `/returns` with body `{"transfer_id":"<id>","reason":"customer request"}` → state=`Returned`, `reversal_fee=30`
- [ ] Show ledger entry created (reversal + fee)

### 10. Idempotency
- [ ] POST deposit with `X-Idempotency-Key: demo-key-1`
- [ ] Repeat same request → `X-Idempotency-Replayed: true` header

### 11. Settlement
- [ ] POST several clean deposits (e.g. ACC-001) so some are in `FundsPosted`
- [ ] (Ensure logged in) POST `/settlement/trigger`
- [ ] Show batch_id, total_count, total_amount, optional `after_eod_cutoff`
- [ ] Show `settlements/settlement-*.json` file created in server working directory

### 12. Ledger View
- [ ] `GET /ledger` → list of ledger entries
- [ ] `GET /accounts/ACC-001/balance` → balance for account ACC-001

## Post-Demo
- [ ] Run `go test ./...` to show all tests passing
- [ ] Show `docs/architecture.md` state machine diagram
