# Checkstream Submission

## Project name

Checkstream (Apex Mobile Deposit) — mobile check deposit backend.

## Summary (3–5 sentences)

We built a minimal end-to-end mobile check deposit system in Go: deposit submission → Vendor stub (IQA/MICR/OCR) → Funding Service business rules → ledger posting → operator review for flagged items → EOD settlement (X9-like JSON) and return/reversal with $30 fee. Design choices: SQLite for zero-config demo, in-process vendor stub with `config/scenarios.json` for deterministic scenarios, and structured DEPOSIT_TRACE logs for observability. Key trade-offs: JSON settlement files instead of binary X9 ICL (readable for demo; production would use moov-io/x9 or a licensed encoder), and log-only investor notification / batch-time settlement ack for MVP.

## How to run (copy-paste commands)

```bash
go mod tidy
make test     # all tests pass
make dev      # start server on :8080
# In another terminal:
bash scripts/demo.sh
```

Or with Docker: `docker-compose up`

## Test / eval results

- **Report:** `reports/test_report.txt` (in repo; regenerate with `make test-report` if needed)
- All tests pass: `go test ./...` — happy path, all vendor stub scenarios, business rules, state machine, reversal with fee, settlement file validation

## With one more week, we would

- Add JWT/OAuth for deposit and return APIs (customer-facing auth)
- Persist settlement files to S3 and add a real settlement-bank callback or polling for ack
- Implement idempotency key expiration (e.g. 24h TTL)
- Add a dedicated investor notification channel (email or push) on return
- Optional: binary X9 ICL export via moov-io/x9 for bank submission

## Risks and limitations

See **`docs/risks_limitations.md`** for the full note (no compliance or regulatory claims). Summary: partial auth (operator-only today), base64 images in DB, SQLite single-writer limit, no vendor retry/DLQ, investor notification and settlement bank ack are stubbed, JSON not spec X9 ICL, redacted logs documented (synthetic data only).

## How should ACME evaluate production readiness?

- **Security:** Add auth (JWT/OAuth) and RBAC for all endpoints; move images to object storage; secrets in a vault.
- **Scale:** Migrate to PostgreSQL; add retry/circuit breaker for vendor; consider async ledger posting and read replicas.
- **Compliance:** Integrate real X9 ICL and bank ack; implement idempotency expiry and full audit retention; formalize investor notification and PII redaction in logs.
- **Operational:** Run `GET /health/settlement` in monitoring; alert on `unsettled_funds_posted` after EOD; add structured metrics (e.g. deposit volume, queue depth).

## Architecture and decisions

- **Architecture:** `docs/architecture.md` — service boundaries, state machine, API routes
- **Decision log:** `docs/decision_log.md` — settlement format, stub design, state machine rationale
- **Short write-up:** `docs/short_writeup.md` — ≤1 page summary
