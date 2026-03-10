# Task 010: Observability & Monitoring

## Goal

Implement observability and monitoring per [requirements.md](../../../requirements.md) § Observability & Monitoring.

## Deliverables

- [ ] **Per-deposit decision trace:** Structured trace per deposit: inputs → Vendor response → business rules applied → operator actions → settlement status
  - Store in transfer record or separate `deposit_traces` table; or emit structured log entries with `transfer_id` for correlation
- [ ] **Differentiate deposit sources in logs:** Tag logs with source (e.g. `source=api`, `source=demo`, `account_id` prefix) for debugging
- [ ] **Monitor for missing or delayed settlement files:** Health check or scheduled job that flags when expected EOD settlement file is missing or late; log warning
- [ ] **Redacted logs:** No real PII in any scenario; redact account numbers, MICR, amounts in log output (or use synthetic data only per requirements)

## Notes

- Decision trace can be JSON field on transfer or append-only log; must be queryable for debugging
- Settlement monitoring: e.g. `GET /health/settlement` that returns OK if today's file exists or cutoff not yet passed
- Redaction: use placeholder or hash for sensitive fields; document in decision log

## Verification

- [ ] Trace available for a deposit through full lifecycle
- [ ] Logs include source tags
- [ ] Settlement health check returns expected state
- [ ] No PII in log output (manual audit)
