# Risks & Limitations

## Security Risks

### R-001: Partial Authentication
- **Severity:** Medium (demo); High (production)
- **Description:** Deposit, returns, ledger, health, and vendor endpoints are unauthenticated. Operator and settlement endpoints (queue, approve, reject, audit, settlement trigger, clock) require operator login via cookie session (`POST /operator/login` or `POST /operator/guest`). In production, all endpoints should use JWT/OAuth2 with role-based access control; customer-facing deposit/return APIs need their own auth.
- **Mitigation:** Add token validation and role checks for production; keep operator middleware pattern for RBAC.

### R-002: Base64 Images Stored in DB
- **Severity:** Medium
- **Description:** Check images are stored as base64 text in SQLite. For real deployments, these should go to encrypted object storage (S3 with SSE).
- **Mitigation:** Replace with presigned S3 URLs and store only references.

### R-003: SQLite Single-Writer Limitation
- **Severity:** Medium
- **Description:** SQLite WAL mode allows concurrent reads but only one writer at a time. Under high deposit load, writes will queue.
- **Mitigation:** Migrate to PostgreSQL for production.

## Operational Risks

### R-004: No Retry Logic for Vendor Calls
- **Severity:** Medium
- **Description:** The vendor stub is in-process; a real vendor integration needs retry with exponential backoff and circuit breaker.

### R-005: Settlement File Not Persisted to S3
- **Severity:** Low (demo)
- **Description:** Settlement files are written to the local filesystem. In production these should be uploaded to secure storage and transmitted via SFTP/HTTPS to the bank.

### R-006: No Dead Letter Queue
- **Severity:** Medium
- **Description:** Failed transitions or ledger errors are logged but not queued for retry. A message queue (SQS/Kafka) should handle failure cases.

## Functional Limitations

### L-001: No Real X9 ICL Format
- JSON settlement files are not spec-compliant X9 ICL binary format. A licensed encoder is needed for bank submission.

### L-002: No Duplicate Detection on Check Serial Number
- Duplicate detection uses vendor `transaction_id`. A production system should also check MICR routing+account+check_number combinations.

### L-003: No EOD Cutoff Enforcement at Deposit Time
- Deposits submitted after 6:30 PM CT are accepted; they simply appear in the next settlement batch. Explicit user notification of next-day posting is not implemented.
- Settlement trigger response includes `after_eod_cutoff` for observability; settlement proceeds regardless.

### L-004: No Idempotency Key Expiration
- Cached idempotency responses are stored indefinitely. Production should expire keys after 24 hours per RFC guidance.

### L-005: Return from Completed Not Implemented
- The return service accepts transfers in either `FundsPosted` or `Completed`, but the state machine in `internal/transfer/state.go` only allows `FundsPosted → Returned`. A transfer in `Completed` cannot transition to `Returned`; `ProcessReturn` will fail with an invalid-transition error. Only returns from `FundsPosted` are supported until `validTransitions` includes `Completed → Returned`.
