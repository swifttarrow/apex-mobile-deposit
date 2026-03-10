# Task 006: Session Validation & Account Eligibility

## Goal

Validate investor session and account eligibility before applying funding rules. Per [requirements.md](../../../requirements.md): "Validate investor session and account eligibility."

## Deliverables

- [ ] Session validation: ensure request includes valid account identifier; reject if missing or invalid
- [ ] Account eligibility: account must exist in client/correspondent config and be eligible for deposits
- [ ] For MVP: eligibility = account_id present in omnibus/config map; document simplification in decision log if needed
- [ ] Return 401/403 with clear message when session or eligibility check fails

## Notes

- In production, session would validate JWT/session token; for stub, validate account_id presence and config lookup
- Eligibility could extend to: account not closed, account type allows check deposits, etc.
- Document trade-offs in docs/decision_log.md (e.g. "Session validation stubbed to config lookup for MVP")

## Verification

```bash
go test ./internal/funding/...
```

- Request with unknown account_id → rejected
- Request with valid account_id in config → proceeds to business rules
