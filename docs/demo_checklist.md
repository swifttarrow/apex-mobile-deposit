# Demo Video Checklist

## Pre-Demo Setup
- [ ] `go mod tidy` completed successfully
- [ ] `go test ./...` all passing
- [ ] `make build` produces binary
- [ ] Server running: `make dev`
- [ ] Terminal with `jq` installed for pretty output

## Demo Flow

### 1. Health Check
- [ ] `GET /health` returns `{"status":"ok"}`

### 2. Clean Pass Deposit
- [ ] POST to `/deposits` with `ACC-001`, $150
- [ ] Response: `201 Created`, state=`FundsPosted`
- [ ] Show transaction_id and MICR data in response

### 3. GET Deposit Status
- [ ] `GET /deposits/:id` for the above transfer
- [ ] Show state=`FundsPosted`

### 4. IQA Failures
- [ ] POST with `ACC-IQA-BLUR` → `422 Unprocessable Entity`, reason=blur
- [ ] POST with `ACC-IQA-GLARE` → `422 Unprocessable Entity`, reason=glare

### 5. Operator Workflow (MICR Fail)
- [ ] POST with `ACC-MICR-FAIL` → `202 Accepted`, state=`Analyzing`
- [ ] `GET /operator/queue` → shows flagged transfer
- [ ] POST `/operator/approve` → state=`FundsPosted`

### 6. Amount Mismatch → Operator Rejects
- [ ] POST with `ACC-MISMATCH` → `202 Accepted`, state=`Analyzing`
- [ ] POST `/operator/reject` → state=`Rejected`

### 7. Over Deposit Limit
- [ ] POST with `ACC-OVER-LIMIT`, $6000 → `422`, over limit error

### 8. Retirement Account
- [ ] POST with `ACC-RETIRE-001`, $1000
- [ ] Response: `201`, `contribution_type=individual`

### 9. Return/Reversal
- [ ] POST deposit with `ACC-001`, $500 → `201 FundsPosted`
- [ ] POST `/returns` → state=`Returned`, `reversal_fee=30`
- [ ] Show ledger entry created

### 10. Idempotency
- [ ] POST deposit with `X-Idempotency-Key: demo-key-1`
- [ ] Repeat same request → `X-Idempotency-Replayed: true` header

### 11. Settlement
- [ ] POST several clean deposits
- [ ] POST `/settlement/trigger`
- [ ] Show batch_id, total_count, total_amount
- [ ] Show `settlements/settlement-*.json` file created

### 12. Ledger View
- [ ] `GET /ledger` → all entries
- [ ] `GET /accounts/ACC-001/balance` → balance amount

## Post-Demo
- [ ] Run `go test ./...` to show all tests passing
- [ ] Show `docs/architecture.md` state machine diagram
