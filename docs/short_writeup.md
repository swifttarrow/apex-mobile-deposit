# Short Write-Up: Architecture, Stub Design, State Machine, Risks

## Architecture choices

The system is a single Go service with a clear HTTP layer, orchestration in `api/deposits.go`, and separate modules for transfer persistence, vendor stub, funding rules, and ledger. We use **SQLite** (WAL) for zero-config local development and full ACID; production would migrate to PostgreSQL. The **vendor integration is in-process** (no outbound HTTP) so tests and demos run without network; scenario selection is deterministic via `config/scenarios.json` and optional `X-Test-Scenario` header. **Settlement output** is X9-like JSON written to `settlements/` (not binary X9 ICL); production would use moov-io/x9 or a licensed encoder. **Operator auth** is cookie-based (login/guest); deposit and return endpoints are unauthenticated in the demo.

## Vendor Service stub design

The stub supports seven selectable outcomes: **IQA fail (blur)**, **IQA fail (glare)**, **MICR read failure** (flagged), **duplicate** (reject), **amount mismatch** (flagged), **IQA pass**, and **clean pass** (MICR data + transaction ID). Selection is by account ID prefix in `config/scenarios.json` or by request header `X-Test-Scenario`, so evaluators can exercise every path without code changes. Responses are deterministic; the stub is independently testable and documented in the README test-accounts table.

## State machine rationale

We use eight states: **Requested → Validating → Analyzing → Approved → FundsPosted → Completed**, with **Rejected** and **Returned** as terminal branches. Validating handles vendor result (fail → Rejected, flagged/pass → Analyzing). Business rules run in Analyzing; only then can we transition to Approved and post the ledger. We allow **Completed → Returned** so returns after settlement are supported. Transitions are enforced in code; invalid moves return clear errors and never update the DB.

## Risks and limitations

- **Auth:** Only operator endpoints are protected; deposit/return/ledger are open. Production needs JWT/OAuth and RBAC.
- **Data:** Check images are base64 in SQLite; settlement files are local. Production: object storage and secure transfer to the bank.
- **Vendor:** Stub is in-process; real integration needs retry and circuit breaker.
- **Settlement:** We set `SettlementAckAt` at batch generation; there is no bank callback. Investor notification on return is log-only.
- **Format:** JSON settlement is not spec X9 ICL; no compliance claim. Logs use synthetic data only; production would require explicit PII redaction.

Full list: **`docs/risks_limitations.md`**.
