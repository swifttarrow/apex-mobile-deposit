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
2. Run the demo script: `bash scripts/demo.sh`

Or use the curl commands in `scripts/demo.sh` individually.

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
| POST | /vendor/validate | Vendor stub endpoint |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `checkstream.db` | SQLite database path |

See `.env.example` for all variables.
