# Checkstream: Architecture Research & Pre-Search

*Research output per `agent/prompts/research.md` on `docs/prd.md`*

---

## Phase 1 — System Understanding

### Project Summary (5–10 Bullets)

- **System purpose:** End-to-end mobile check deposit pipeline for brokerage platforms—capture, validation, compliance gating, operator review, ledger posting, settlement, and return/reversal handling.
- **Core user interaction loop:** Investor submits check images (front/back) + amount + account → Vendor validates (IQA/MICR/OCR) → Funding enforces business rules → Ledger posts → Settlement file generated → EOD cutoff respected.
- **Primary technical components:** Vendor API stub, Funding Service, ledger, transfer state machine, operator review queue, settlement file generator (X9 ICL or equivalent), return/reversal handler.
- **Most difficult engineering challenges:** Correct transfer state machine with valid transitions only; atomic multi-step writes (ledger + transfer state); concurrent updates (return during settlement); EOD cutoff enforcement; idempotency for deposit submission and return processing.
- **Expected scale:** MVP targets modest load (deposits/minute not specified); production projections: 100–100K users with $0.10/check validation.
- **Latency / performance:** Vendor stub < 1s; ledger posting < 5s from approval; settlement file < 5s from EOD trigger; operator queue < 1s; state transitions queryable within 1s.
- **Major external dependencies:** Vendor Service (stubbed), Settlement Bank (stubbed via X9 file), omnibus account lookup.
- **AI/LLM usage:** None. Operational cost tracking only.

---

### Hard Problems

| # | Problem | Why It's Difficult |
|---|---------|--------------------|
| 1 | **Transfer state machine correctness** | 8 states, 10+ transitions; invalid transitions must be rejected; concurrent events (return during settlement) require clear ordering and conflict resolution. |
| 2 | **Atomic multi-step writes** | Ledger posting + transfer state update must succeed or fail together; partial failures (ledger posted, settlement failed) require compensation logic. |
| 3 | **Vendor stub configurability** | 7+ differentiated scenarios must be selectable without code changes; evaluators need deterministic triggers (test account, header, config). |
| 4 | **Settlement file format** | X9 ICL is complex (File Header, Cash Letter, Bundle, Check Detail, Image View, etc.); minimal viable structure must map to real bank expectations. |
| 5 | **EOD cutoff enforcement** | 6:30 PM CT cutoff; late submissions roll to next business day; batch boundaries and settlement file generation timing must align. |
| 6 | **Idempotency** | Deposit submission and return processing must be idempotent to avoid double-posting; requires idempotency keys and deduplication. |
| 7 | **Operator workflow data model** | Review queue needs images, MICR data, risk scores, amount comparison; search/filter by date, status, account, amount; audit logging for all actions. |
| 8 | **Return/reversal ledger entries** | Reversal must debit investor, credit omnibus, deduct $30 fee; must handle duplicate return notifications idempotently. |

---

## Phase 2 — Architecture Exploration

### Architecture A: Monolithic Single Process

**High-Level Design**

- Single process (Go or Java) hosting all components: API, Vendor stub, Funding Service, ledger, operator workflow, settlement engine.
- SQLite for transfers, ledger entries, operator actions, audit logs.
- In-process Vendor stub; no network hop for validation.
- Minimal UI (React/Vue/Svelte) or CLI for deposit submission and operator review.
- Settlement file written to disk; EOD trigger via cron or manual endpoint.

**Key Components**

- **API layer:** REST endpoints (deposit submit, status, operator approve/reject, settlement trigger).
- **Vendor stub:** In-process module; config-driven response selection.
- **Funding Service:** Business rules, account resolution, ledger posting.
- **Ledger:** SQLite tables for MOVEMENT entries.
- **Transfer state machine:** Persisted in SQLite; enforced in application code.
- **Operator queue:** Query layer over flagged transfers.
- **Settlement engine:** Batch query + X9/JSON file generation.

**Data Flow**

```
User → API → Vendor Stub → Funding Service → Ledger + Transfer DB
                                    ↓
                            Operator Queue (if flagged)
                                    ↓
                            Settlement Engine → File
```

**Strengths**

- Fastest to build; no service boundaries or network latency.
- Single transaction scope for ledger + transfer updates.
- One-command setup trivial (single binary + SQLite).
- Easiest to debug and test locally.

**Weaknesses**

- No horizontal scaling; single point of failure.
- Vendor stub in-process means no realistic simulation of network failures.
- All components share same process; one bug can crash everything.

**Best Use Case**

- MVP and one-week sprint; prioritize correctness and speed over scale.

---

### Architecture B: Service-Oriented (Stub + API + Worker)

**High-Level Design**

- Vendor stub as separate HTTP service (Docker container).
- Main API service: deposit submission, Funding, ledger, operator workflow.
- Optional worker for settlement file generation (scheduled or triggered).
- PostgreSQL for production-grade persistence; SQLite for local dev.
- Queue (Redis, in-memory, or DB-backed) for async settlement.

**Key Components**

- **Vendor stub service:** Standalone HTTP server; configurable responses.
- **API service:** REST, Funding logic, ledger, state machine.
- **Worker:** Settlement batch job; return notification processor.
- **PostgreSQL:** Transfers, ledger, operator actions, audit logs.
- **Queue:** Settlement jobs, return notifications.

**Data Flow**

```
User → API → Vendor Stub (HTTP) → Funding → Ledger + Transfer DB
                                        ↓
                                Operator Queue
                                        ↓
                                Worker → Settlement File
```

**Strengths**

- Realistic Vendor integration pattern; stub can simulate timeouts, failures.
- Worker decouples settlement from request path; EOD job can run independently.
- PostgreSQL supports transactions, better concurrency.

**Weaknesses**

- More moving parts; Docker Compose complexity.
- Network latency between API and stub (though minimal for MVP).
- Overkill for 24-hour MVP gate.

**Best Use Case**

- When evaluators want to test stub as a real service; when scaling beyond MVP.

---

### Architecture C: Event-Driven (Async Pipeline)

**High-Level Design**

- Deposit submission produces event; async pipeline processes validation → funding → ledger → settlement.
- Event store or message queue (Kafka, Redis Streams, or DB events) for transfer lifecycle.
- Operator actions emit events; settlement engine consumes Approved events.
- Event sourcing optional for audit trail.

**Key Components**

- **Event bus:** Redis Streams, Kafka, or DB-backed event log.
- **Deposit handler:** Publishes DepositSubmitted.
- **Validation handler:** Calls Vendor stub; publishes Validated/Rejected/Flagged.
- **Funding handler:** Applies business rules; publishes Approved/Rejected.
- **Ledger handler:** Posts MOVEMENT; publishes FundsPosted.
- **Settlement handler:** Batches by EOD; generates file; publishes Completed.

**Data Flow**

```
User → API → DepositSubmitted event
                ↓
        Validation Handler → Vendor Stub → Validated/Rejected/Flagged
                ↓
        Funding Handler → Approved/Rejected
                ↓
        Ledger Handler → FundsPosted
                ↓
        Settlement Handler (EOD) → Completed
```

**Strengths**

- Natural fit for async flows; easy to add handlers.
- Clear audit trail via event log.
- Scales by adding consumers.

**Weaknesses**

- Highest complexity; event ordering, exactly-once semantics.
- Overkill for MVP; 24-hour timeline makes this risky.
- Debugging async flows harder.

**Best Use Case**

- Production at scale; not recommended for one-week sprint.

---

## Phase 3 — Deep Technical Decisions

### Decision 1: Backend Language

| Option | Pros | Cons |
|--------|------|------|
| **Go** | Fast compile, single binary, strong stdlib, good for services | Less mature ecosystem for some financial libs |
| **Java** | Mature, Spring Boot, strong typing | Heavier runtime, slower startup |
| **Other** | — | PRD constrains to Go or Java |

**Tradeoffs:** Go favors speed of development and deployment; Java favors enterprise patterns. For one-week sprint, Go's simplicity and single-binary deployment win.

**Recommendation:** **Go.** Rationale: Single binary, `go mod`, fast iteration, Docker image small. Moov has Go X9 ICL library (`moov-io/imagecashletter`).

---

### Decision 2: Vendor Stub — In-Process vs Separate Service

| Option | Pros | Cons |
|--------|------|------|
| **In-process** | No network, single deploy, fastest MVP | Doesn't simulate real integration |
| **Separate service** | Realistic HTTP client, timeout simulation | Extra container, more setup |

**Tradeoffs:** In-process = faster to build, one-command setup simpler. Separate = better for integration testing, evaluator can hit stub directly.

**Recommendation:** **In-process module with configurable router.** Rationale: PRD says "stub in same process or separate service"; for MVP, in-process wins. Expose stub as HTTP handler on same server (e.g. `/vendor/validate`) so it can be called like a real service; keeps option to split later.

---

### Decision 3: Stub Response Selection Mechanism

| Option | Pros | Cons |
|--------|------|------|
| **Test account prefix** | `ACC-IQA-BLUR`, `ACC-MICR-FAIL` — intuitive | Requires account in request |
| **Request header** | `X-Test-Scenario: clean_pass` — flexible | Header must be documented |
| **Config file** | `scenarios.json` maps inputs to outputs | Config must be loaded, editable |
| **Combination** | Support multiple triggers | More code paths |

**Tradeoffs:** Test account is PRD example; header is common in APIs; config allows non-engineers to change. Evaluators need "different inputs produce different stub responses without code changes."

**Recommendation:** **Config file + test account override.** Rationale: `scenarios.json` maps account prefixes to scenario names; fallback to `X-Test-Scenario` header if present. Document in README. Covers all 7+ scenarios deterministically.

---

### Decision 4: Data Store — SQLite vs PostgreSQL

| Option | Pros | Cons |
|--------|------|------|
| **SQLite** | No server, single file, trivial setup | Single writer, limited concurrency |
| **PostgreSQL** | Production-grade, transactions, scaling | Requires DB server, Docker |

**Tradeoffs:** SQLite sufficient for MVP scale; PostgreSQL needed for multi-instance or high concurrency.

**Recommendation:** **SQLite for MVP.** Rationale: One-command setup, no external DB. Use `PRAGMA busy_timeout` and transactions. Document migration path to PostgreSQL (schema compatible).

---

### Decision 5: Settlement File Format — X9 ICL vs Structured JSON

| Option | Pros | Cons |
|--------|------|------|
| **X9 ICL (real)** | Industry standard, Moov lib exists | Complex, many record types |
| **Structured JSON** | Simple, human-readable, PRD allows | Not bank-ready |
| **Minimal X9 subset** | Real format, minimal records | Must implement File Header, Cash Letter, Bundle, Check Detail, Image View, Control |

**Tradeoffs:** PRD says "X9 ICL or structured equivalent." JSON is faster; X9 demonstrates real integration. Moov `imagecashletter` supports X9.100-187.

**Recommendation:** **Use Moov `moov-io/imagecashletter` for X9 ICL.** Rationale: Real format, Go library handles record structure. MVP needs: File Header (01), Cash Letter Header (10), Bundle Header (20), Check Detail (25), Check Detail Addendum A (26), Image View Detail (50), Image View Data (52), Bundle Control (70), Cash Letter Control (90), File Control (99). Beats hand-rolling JSON that doesn't map to production.

---

### Decision 6: Transfer State Machine Persistence

| Option | Pros | Cons |
|--------|------|------|
| **Single `transfers` table + state column** | Simple, one table | Must enforce transitions in app |
| **State history table** | Full audit of transitions | More tables, more writes |
| **Event log** | Full trace, event sourcing | Overkill for MVP |

**Tradeoffs:** Single table with `state` and `updated_at` is minimal. Transition validation in application layer with DB constraint on valid states.

**Recommendation:** **Single `transfers` table with `state` enum; transition validation in application.** Rationale: Keep schema simple. Use DB CHECK or app-level validation to reject invalid transitions. Log state changes to audit table for "who, what, when."

---

### Decision 7: Idempotency Strategy

| Option | Pros | Cons |
|--------|------|------|
| **Idempotency key in request** | Client sends key; server dedupes | Client must generate |
| **Transfer ID from first request** | Server generates; duplicate request returns same ID | Need to define "same" request |
| **Idempotency table** | Store key → result; lookup before processing | Extra table, TTL for cleanup |

**Tradeoffs:** Deposit submission: client sends `X-Idempotency-Key` or similar; server stores key → transfer_id; duplicate returns existing. Return processing: return notification includes check/transaction ID; dedupe by that ID.

**Recommendation:** **Idempotency key header for deposit; transaction/check ID for returns.** Rationale: Deposit: `X-Idempotency-Key: uuid`; store in `idempotency_keys` (key, transfer_id, created_at); 24h TTL. Return: use MICR/transaction ID as idempotency key; reject if return already processed for that check.

---

## Phase 4 — System Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Checkstream MVP                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  User (CLI / Minimal UI)                                                    │
│       │                                                                     │
│       ▼                                                                     │
│  ┌─────────────────┐                                                        │
│  │   REST API      │  POST /deposits, GET /deposits/:id, GET /transfers     │
│  │   (Go)          │  POST /operator/approve, POST /operator/reject        │
│  └────────┬────────┘  POST /settlement/trigger (EOD)                        │
│           │                                                                 │
│           ├──────────────────────────────────────────────────────────────┐  │
│           │                                                              │  │
│           ▼                                                              ▼  │
│  ┌─────────────────┐                                              ┌─────────────┐
│  │ Vendor Stub     │  (in-process HTTP handler)                   │ Funding     │
│  │ /vendor/validate│  Config: scenarios.json                      │ Service     │
│  │                 │  Triggers: account prefix, X-Test-Scenario   │ - Limits    │
│  └────────┬────────┘                                              │ - Duplicates│
│           │                                                       │ - Ledger    │
│           │  IQA/MICR/OCR result                                  └──────┬──────┘
│           └──────────────────────────────────────────────────────────────┤
│                                                                          │
│           ┌──────────────────────────────────────────────────────────────┘
│           │
│           ▼
│  ┌─────────────────────────────────────────────────────────────────────────┐
│  │  SQLite                                                                 │
│  │  ├── transfers (id, account_id, amount, state, vendor_response, ...)   │
│  │  ├── ledger_entries (movements: To, From, Amount, SubType, ...)        │
│  │  ├── operator_actions (transfer_id, action, operator_id, timestamp)    │
│  │  ├── idempotency_keys (key, transfer_id, expires_at)                   │
│  │  └── check_images (transfer_id, front, back) — or blob path           │
│  └─────────────────────────────────────────────────────────────────────────┘
│           │
│           ▼
│  ┌─────────────────┐     ┌──────────────────┐
│  │ Settlement      │     │ Return/Reversal  │
│  │ Engine          │     │ Handler          │
│  │ - EOD cutoff    │     │ - Debit + $30    │
│  │ - X9 ICL gen   │     │ - State→Returned │
│  │ - Moov lib      │     │ - Idempotent     │
│  └────────┬────────┘     └──────────────────┘
│           │
│           ▼
│  Settlement File (X9 ICL) → disk / stub Settlement Bank
│
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 5 — Risks & Unknowns

### Top Technical Risks

| # | Risk | Mitigation |
|---|------|------------|
| 1 | **State machine race:** Return arrives during FundsPosted → Completed | Define strict transition rules; return can only apply to Completed transfers; use DB transactions and locking. |
| 2 | **Partial failure:** Ledger posted but settlement file fails | Log and retry; settlement is idempotent (batch by date); add reconciliation job. |
| 3 | **Vendor stub timeout:** Simulated timeout mid-validation | Stub can support `X-Test-Scenario: timeout`; API must handle timeout → Retry or move to Rejected/Flagged per business rule. |
| 4 | **X9 ICL format errors:** Generated file rejected by bank | Use Moov library; validate against sample files; document minimal record set. |
| 5 | **EOD cutoff edge cases:** Deposit at 6:30:00 PM CT | Define cutoff as exclusive (>= 6:30 PM CT rolls); use timezone-aware logic (America/Chicago). |
| 6 | **Operator auth:** No auth in MVP | Document as limitation; add placeholder middleware for API key or session. |
| 7 | **Image storage:** Check images in DB vs blob store | SQLite BLOB or file path; for MVP, base64 in JSON or file path. Keep under 2MB per deposit. |

### Unknowns Worth Researching

- **Omnibus account lookup:** PRD says "looked up via client config"—exact schema for account → omnibus mapping.
- **Settlement Bank acknowledgment:** How to stub "settlement confirmed"; simple flag or async callback?
- **X9 Image View Data:** TIFF format requirements; bitonal vs grayscale; Moov lib defaults.

### Fast Experiments

1. **Moov imagecashletter:** Create minimal X9 file with one check; verify record order and field lengths.
2. **State machine table:** Encode valid transitions in a table; unit test all (state, event) → new state.
3. **Stub config:** Implement `scenarios.json` with 3 scenarios; verify request routing in < 30 min.
4. **SQLite transaction:** Benchmark ledger + transfer update in single transaction; confirm atomicity.
5. **EOD cutoff:** Unit test time comparisons for 6:30 PM CT with `time.LoadLocation("America/Chicago")`.

---

## Phase 6 — MVP Architecture

**Recommendation: Monolithic Go service with SQLite (Architecture A).**

### What to Build First

1. **Transfer state machine** — Schema, valid transitions table, persistence.
2. **Vendor stub** — Config-driven, 7+ scenarios, deterministic.
3. **Funding Service** — Limits, duplicates, ledger posting.
4. **Deposit submission flow** — API → stub → funding → state.
5. **Operator workflow** — Queue, approve/reject, audit log.
6. **Settlement file** — Moov X9 ICL, EOD trigger.
7. **Return/reversal** — Notification endpoint, reversal posting, fee.

### What to Fake or Defer

- **Real Vendor API:** Stub only.
- **Real Settlement Bank:** Write file to disk; optional stub endpoint that "acknowledges."
- **Omnibus lookup:** Hardcode or simple config map (account_id → omnibus_id).
- **Operator auth:** Defer; document as MVP limitation.
- **Production observability:** Basic logging; defer metrics/traces.

### What to Simplify

- **Image storage:** File paths or base64 in DB; no S3/blob store.
- **Search/filter:** Simple SQL WHERE; no Elasticsearch.
- **EOD trigger:** Manual `POST /settlement/trigger`; cron optional.
- **Return notification:** `POST /returns` with transfer ID; no webhook simulation.

### Locked Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Single binary, fast iteration, Moov X9 lib |
| Stub location | In-process | Fastest MVP; expose as HTTP handler |
| Stub config | scenarios.json + account/header | Deterministic, no code changes |
| Database | SQLite | One-command setup, no external deps |
| Settlement format | X9 ICL via Moov | Real format, production path |
| State machine | Single table + app validation | Simple, audit via operator_actions |
| Idempotency | Header (deposit) + check ID (return) | Prevents double-posting |

> *"A simple deposit pipeline with correct state transitions beats a complex one with broken gating."*

---

## Phase 7 — Iteration Mode

**Confirmed (user input):**

1. **Constraints:** Optimize for **speed**
2. **Architecture:** **Monolithic Go + SQLite** — accepted
3. **Deep dive:** Deferred; user will run other commands

---

## Research Output: Decisions + Rationale

Research is complete. Each decision above includes:

- **Decision** — Concrete, actionable choice.
- **Rationale** — Constraints, tradeoffs, optimization target.

Ready for handoff to Planning. Next step: Create implementation plan in `thoughts/plans/`.
