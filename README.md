# Checkdepot Mobile Check Deposit

A Go backend service implementing mobile check deposit with state machine-based transfer lifecycle, vendor check analysis, operator review workflow, and settlement processing.

**Pre-Search:** Architecture research and constraints are documented in [docs/research/checkstream-prd-research.md](docs/research/checkstream-prd-research.md).

## Architecture Overview

```
POST /deposits
  → Create Transfer (Requested) + enqueue async deposit job
  → Returns 202 Accepted immediately
  → Worker: Vendor IQA Check (Validating)
  → If fail/reject: → Rejected
  → If flagged: → Analyzing (awaits operator)
  → If pass: → Analyzing → Approved
  → Business Rules: session, eligibility, limit, duplicate
  → Post Ledger Entry
  → → FundsPosted

Operator Workflow:
  GET /operator/queue → list Analyzing transfers
  POST /operator/approve → Analyzing → Approved → FundsPosted
  POST /operator/reject  → Analyzing → Rejected

Settlement:
  POST /settlement/trigger → batch FundsPosted → Completed

Returns:
  POST /returns → FundsPosted/Completed → Returned + reversal entry
```

**Stubs (MVP):** Investor notification on return is log-only; settlement bank acknowledgment is the batch-generation timestamp (no bank callback). See [docs/risks_limitations.md](docs/risks_limitations.md). **Logging:** No real PII is logged; all data is synthetic. DEPOSIT_TRACE lines include optional `source` (e.g. `mobile`, `api`) for debugging.

## Transfer States

```
Requested → Validating → Analyzing → Approved → FundsPosted → Completed
                  ↓            ↓                              ↓
               Rejected     Rejected                       Returned
```

## Setup

### Prerequisites
- Go 1.23+
- gcc (for CGO/SQLite)

### Install & Run

```bash
git clone <repo>
cd apex-mobile-deposit
go mod tidy
make dev
```

The server starts on `:8080`.

### Scenario Showcase UI

Visit **http://localhost:8080/** for the operator dashboard. Visit **http://localhost:8080/sandbox/** to see a visual reference of all user story scenarios. The UI shows:
- **Case** — Scenario name and ID (e.g. US-1.1, US-2.1)
- **Flow** — Step-by-step state transitions and API calls
- **Pass/Fail** — Run scenarios against the live API and see results with full response bodies

Requires the server to be running. Use the API base field if the UI is served from a different origin.

### Run Tests

```bash
make test
```

### Build

```bash
make build
./bin/checkstream
```

### Docker

```bash
docker-compose up
```

## Test Accounts

| Account ID | Scenario | Expected Result |
|------------|----------|-----------------|
| `ACC-001` | Clean pass | `POST /deposits` returns 202; final state `FundsPosted` |
| `ACC-IQA-BLUR` | IQA blur fail | `POST /deposits` returns 202; final state `Rejected` |
| `ACC-IQA-GLARE` | IQA glare fail | `POST /deposits` returns 202; final state `Rejected` |
| `ACC-MICR-FAIL` | MICR read fail | `POST /deposits` returns 202; final state `Analyzing` (flagged) |
| `ACC-DUP-001` | Duplicate | `POST /deposits` returns 202; final state `Rejected` |
| `ACC-MISMATCH` | Amount mismatch | `POST /deposits` returns 202; final state `Analyzing` (flagged) |
| `ACC-OVER-LIMIT` | Over $5000 limit | `POST /deposits` returns 202; final state `Rejected` |
| `ACC-RETIRE-001` | Retirement account | `POST /deposits` returns 202; final state `FundsPosted` (`contribution_type=individual`) |

## How to Demo

1. Start the server: `make dev`
2. In another terminal, run the **browser demo** (recommended):
   ```bash
   make demo-install   # once: install Node deps + Chromium
   make demo           # runs Playwright: mobile deposits → operator review → settlement
   ```
   To **watch** the demo in a visible browser, run `make demo-headed` (or `cd e2e && npm run demo:headed`). To slow it down, pass `SLOW=500` (milliseconds between actions), e.g. `make demo-headed SLOW=500`, or use `npm run demo:slow` (default 500 ms).
   The demo drives the **mobile app** and **operator dashboard**: it logs in as test user `joe` / `password`, submits deposits, approves flagged items, and runs settlement.

   To demo **settlement batching and EOD rollover** (some deposits before 6:30 PM CT, some after), seed sample data first: `make seed-deposits` (inserts 25 deposits in `FundsPosted` state with `created_at` spread before/after the cutoff), then run settlement from the operator UI.

Alternatively, run the **CLI demo**: `bash scripts/demo.sh` (or use the curl commands in it individually).

## API Endpoints (Core)

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| POST | /deposits | Submit a check deposit |
| GET | /deposits | List deposits |
| GET | /deposits/{id} | Get deposit status |
| POST | /sandbox/process-job | Process one queued deposit job (sandbox/testing helper) |
| POST | /operator/login | Operator login |
| POST | /operator/guest | Guest operator login |
| GET | /operator/queue | List flagged transfers |
| POST | /operator/approve | Approve a flagged transfer |
| POST | /operator/reject | Reject a flagged transfer |
| POST | /settlement/trigger | Trigger EOD settlement |
| POST | /settlement/report | Generate on-demand settlement report |
| GET | /settlement/reports | List settlement reports |
| POST | /returns | Process a check return |
| GET | /ledger | List ledger entries |
| GET | /accounts/{id}/balance | Get account balance |
| GET | /health/settlement | Settlement monitoring (unsettled count, EOD cutoff) |
| POST | /vendor/validate | Vendor stub endpoint |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `checkstream.db` | SQLite database path |
| `PORT` | `8080` | HTTP server port |
| `SESSION_SECRET` | dev default (if unset) | Cookie/session signing secret (set explicitly outside demo) |

See `.env.example` for all variables.

## Submission

### Project name

Checkstream (Apex Mobile Deposit) - mobile check deposit backend.

### Summary (3-5 sentences)

We built a minimal end-to-end mobile check deposit system in Go: deposit submission -> Vendor stub (IQA/MICR/OCR) -> Funding Service business rules -> ledger posting -> operator review for flagged items -> EOD settlement (X9-like JSON) and return/reversal with $30 fee. Design choices: SQLite for zero-config demo, in-process vendor stub with `config/scenarios.json` for deterministic scenarios, and structured `DEPOSIT_TRACE` logs for observability. Key trade-offs: JSON settlement files instead of binary X9 ICL (readable for demo; production would use moov-io/x9 or a licensed encoder), and log-only investor notification / batch-time settlement ack for MVP.

### How to run (copy-paste commands)

```bash
go mod tidy
make test
make dev
# In another terminal:
bash scripts/demo.sh
```

Or with Docker: `docker-compose up`

### Test / eval results

- Report: `reports/test_report.txt` (regenerate with `make test-report`)
- All tests pass with `go test ./...` (happy path, vendor scenarios, business rules, state machine, reversal fee, settlement validation)

### With one more week, we would

- Add JWT/OAuth for deposit and return APIs
- Persist settlement files to S3 and add real settlement-bank ack callback/polling
- Implement idempotency key expiration (for example, 24h TTL)
- Add a dedicated investor notification channel (email/push/in-app) for returns
- Optional: binary X9 ICL export via moov-io/x9

### Risks and limitations

See `docs/risks_limitations.md` for the full note (no compliance or regulatory claims). Current MVP limitations include partial auth scope, base64 image storage in DB, SQLite write constraints, no vendor retry/DLQ, stubbed investor notification and settlement ack, and JSON settlement output instead of spec X9 ICL.

### How should ACME evaluate production readiness?

- Security: add auth and RBAC for all endpoints, move images to secure object storage, store secrets in a vault
- Scale: migrate to PostgreSQL, add retry/circuit-breaker for vendor calls, consider async processing
- Compliance: integrate real X9 ICL + bank acknowledgment, formalize retention/redaction controls
- Operational: monitor `GET /health/settlement`, alert on delayed settlement, add metrics for volume and queue depth

### Architecture and decisions

- Architecture: `docs/architecture.md`
- Decision log: `docs/decision_log.md`
- Short write-up: `docs/short_writeup.md`
