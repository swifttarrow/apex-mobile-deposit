# Checkdepot Mobile Check Deposit

A Go backend service implementing mobile check deposit with state machine-based transfer lifecycle, vendor check analysis, operator review workflow, and settlement processing.

**Pre-Search:** Architecture research and constraints are documented in [docs/research/checkstream-prd-research.md](docs/research/checkstream-prd-research.md).

## Architecture Overview

```
POST /deposits
  → Create Transfer (Requested)
  → Vendor IQA Check (Validating)
  → If fail/reject: → Rejected (422)
  → If flagged: → Analyzing (202, awaits operator)
  → If pass: → Analyzing → Approved
  → Business Rules: limit, duplicate, eligibility
  → Post Ledger Entry
  → → FundsPosted (201)

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
- Go 1.22+
- gcc (for CGO/SQLite)

### Install & Run

```bash
git clone <repo>
cd checkstream
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
| `ACC-001` | Clean pass | 201 FundsPosted |
| `ACC-IQA-BLUR` | IQA blur fail | 422 Rejected |
| `ACC-IQA-GLARE` | IQA glare fail | 422 Rejected |
| `ACC-MICR-FAIL` | MICR read fail | 202 Analyzing (flagged) |
| `ACC-DUP-001` | Duplicate | 422 Rejected |
| `ACC-MISMATCH` | Amount mismatch | 202 Analyzing (flagged) |
| `ACC-OVER-LIMIT` | Over $5000 limit | 422 Rejected |
| `ACC-RETIRE-001` | Retirement account | 201 FundsPosted (contribution_type=individual) |

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

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| POST | /deposits | Submit a check deposit |
| GET | /deposits/:id | Get deposit status |
| GET | /operator/queue | List flagged transfers |
| POST | /operator/approve | Approve a flagged transfer |
| POST | /operator/reject | Reject a flagged transfer |
| POST | /settlement/trigger | Trigger EOD settlement |
| POST | /returns | Process a check return |
| GET | /ledger | List ledger entries |
| GET | /accounts/:id/balance | Get account balance |
| GET | /health/settlement | Settlement monitoring (unsettled count, EOD cutoff) |
| POST | /vendor/validate | Vendor stub endpoint |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `checkstream.db` | SQLite database path |

See `.env.example` for all variables.
