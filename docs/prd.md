# Checkstream

*Building a robust mobile check deposit pipeline with full lifecycle handling for brokerage platforms*

---

## Before You Start: Pre-Search (1–2 hours)

Complete the **Pre-Search appendix** (Section 16) before writing any code. Your Pre-Search output—saved as a reference document from your AI conversation—is part of your final submission. This week emphasizes system design, integration boundaries, and state-machine correctness. Pre-Search forces you to define constraints, explore architectural options, and lock in stack decisions before implementation. Skipping it leads to costly rework.

---

## Background

Mobile check deposit is table stakes for consumer banking and brokerage platforms. Companies like Chase, Schwab, and Fidelity offer investors the ability to photograph checks and deposit them directly into accounts—reducing reliance on mail, branch visits, and manual processing. Under the hood, these systems integrate with vendor services for image quality assessment (IQA), MICR extraction, OCR, and duplicate detection; route validated deposits through funding middleware that enforces business rules and posts to ledgers; and settle with banks via X9 ICL files. You must build a minimal end-to-end version of this pipeline: capture, validation, compliance gating, operator review, ledger posting, settlement, and return/reversal handling. The core technical challenge is orchestrating multiple subsystems (Vendor API stub, Funding Service, ledger, operator workflow, settlement engine) with a correct transfer state machine and full audit trail.

---

## Gate

**Gate: Project completion required for Austin admission.**

---

## Project Overview

One-week sprint with three deadlines:

| Checkpoint | Deadline | Focus |
|------------|----------|-------|
| Pre-Search | Before coding | Constraints, architecture, stack decisions |
| MVP | Tuesday (24 hrs) | Hard gate: all MVP items complete |
| Early Submission | Friday (4 days) | Full feature set, tests, docs |
| Final | Sunday (7 days) | Deployed app, demo video, all deliverables |

---

## MVP Requirements (24 Hours)

**Hard gate. All items required to pass:**

- ☐ Deposit submission endpoint or UI accepts check images (front/back), amount, and account identifier
- ☐ Vendor Service stub returns differentiated responses (IQA pass, IQA fail blur, IQA fail glare, MICR failure, duplicate, amount mismatch, clean pass) selectable via config or request parameters
- ☐ Funding Service enforces deposit limit ($5,000 MVP cap), contribution defaults, and duplicate detection
- ☐ Transfer state machine implements: Requested → Validating → Analyzing → Approved → FundsPosted → Completed (happy path) and Requested → Validating → Rejected / Returned (non-happy paths)
- ☐ Ledger posting creates correct MOVEMENT entries (To/From accounts, amount, SubType DEPOSIT, Transfer Type CHECK)
- ☐ Operator review queue displays flagged deposits with images, MICR data, and approve/reject controls; actions are logged
- ☐ Settlement file (X9 ICL or structured equivalent) generated with MICR data, images, amounts; EOD cutoff (6:30 PM CT) enforced
- ☐ Return/reversal path: bounced check triggers reversal posting with $30 fee; transfer moves to Returned
- ☐ One-command setup (`make dev` or `docker compose up`) and README with setup instructions
- ☐ Deployed and publicly accessible

*A simple deposit pipeline with correct state transitions beats a complex one with broken gating.*

---

## Core Technical Requirements

### Feature: Deposit Capture & Vendor Integration

| Feature | Requirements |
|---------|--------------|
| Mobile deposit simulation | Endpoint or UI accepts front image, back image, deposit amount, account identifier |
| Vendor API stub | Accepts image payloads; returns structured validation results (IQA, MICR, OCR, duplicate, amount) |
| Response differentiation | Stub returns at least 7 distinct scenarios; selection deterministic (test account, header, or config) |
| Re-submission on IQA fail | Actionable error messages (blur, glare) enabling retake |
| Clean pass | Returns extracted MICR data, amounts, transaction ID |

### Feature: Funding Service & Ledger

| Feature | Requirements |
|---------|--------------|
| Session & account validation | Resolve account identifiers to internal account/routing numbers |
| Business rules | Deposit limit ($5,000), contribution type defaults, duplicate detection |
| Ledger posting | Create transfer with To AccountId (investor), From AccountId (omnibus), Type MOVEMENT, Memo FREE, SubType DEPOSIT, Transfer Type CHECK, Currency USD |
| State machine | All 8 states (Requested, Validating, Analyzing, Approved, FundsPosted, Completed, Rejected, Returned) with valid transitions only |

### Feature: Operator Workflow & Settlement

| Feature | Requirements |
|---------|--------------|
| Review queue | Flagged deposits with check images, MICR data, risk scores, amount comparison |
| Approve/reject | Mandatory action logging (who, what, when) |
| Search & filter | By date, status, account, amount |
| Settlement file | X9 ICL or structured JSON with MICR, images, amounts, batch metadata |
| EOD cutoff | 6:30 PM CT; late submissions roll to next business day |
| Return handling | Reversal debits investor, deducts $30 fee, transitions to Returned, notifies investor |

### Testing Scenarios

We will test:

1. Happy path: submit check → Vendor validates → Funding approves → ledger posted → settlement file generated → Completed
2. IQA fail (blur): submission returns retake prompt; no ledger posting
3. IQA fail (glare): same as blur
4. MICR read failure: deposit flagged for review; operator approves → ledger posted
5. Duplicate detected: deposit rejected; no ledger posting
6. Amount mismatch: deposit flagged; operator approves or rejects
7. Over-limit deposit (>$5,000): rejected by Funding Service
8. Return/reversal: simulated return → reversal posted with fee → transfer Returned
9. EOD cutoff: deposit after 6:30 PM CT rolls to next business day
10. Stub configurability: different inputs produce different stub responses without code changes

### Performance Targets

| Metric | Target |
|--------|--------|
| Vendor stub response | < 1 second |
| Ledger posting latency | < 5 seconds from approval |
| Settlement file generation | < 5 seconds from EOD trigger |
| Operator queue update | Flagged items visible within 1 second |
| State transition propagation | Queryable within 1 second |

---

## Domain-Specific Deep Section: Vendor Service Stub & Transfer State Machine

### Required Capabilities

The Vendor Service stub must support the following response types. Example selection mechanisms:

- **By test account number:** `ACC-IQA-BLUR` → IQA fail blur; `ACC-MICR-FAIL` → MICR read failure; `ACC-DUP-001` → duplicate
- **By request header:** `X-Test-Scenario: clean_pass` → clean pass with extracted MICR
- **By config file:** `scenarios.json` maps account prefixes to response types

| Scenario | Expected Stub Output |
|----------|----------------------|
| IQA Pass | `{ "status": "pass", "iqScore": 0.95 }` — proceed to MICR/OCR |
| IQA Fail (Blur) | `{ "status": "fail", "reason": "blur", "message": "Image too blurry" }` |
| IQA Fail (Glare) | `{ "status": "fail", "reason": "glare", "message": "Glare detected" }` |
| MICR Read Failure | `{ "status": "flagged", "reason": "micr_fail" }` — route to operator |
| Duplicate Detected | `{ "status": "reject", "reason": "duplicate" }` |
| Amount Mismatch | `{ "status": "flagged", "reason": "amount_mismatch", "ocrAmount": 150.00, "enteredAmount": 1500.00 }` |
| Clean Pass | `{ "status": "pass", "micr": {...}, "amount": 150.00, "transactionId": "..." }` |

### Transfer State Machine

Implement at least the following transitions:

| From | To | Trigger |
|------|-----|---------|
| Requested | Validating | Submitted to Vendor |
| Validating | Rejected | IQA fail, duplicate |
| Validating | Analyzing | IQA pass, MICR/OCR complete |
| Analyzing | Rejected | Business rule fail, operator reject |
| Analyzing | Approved | All checks pass, operator approve (if flagged) |
| Approved | FundsPosted | Ledger posting created |
| FundsPosted | Completed | Settlement confirmed |
| FundsPosted | Returned | Return notification received |

### Evaluation Criteria

- Input: deposit with `ACC-IQA-BLUR` → Output: transfer in Rejected, no ledger entry
- Input: deposit with clean pass, under limit → Output: transfer Completed, ledger entry present, settlement file includes deposit
- Input: return notification for completed deposit → Output: reversal posted (amount + $30 fee), transfer Returned

---

## Operational Cost Analysis (Required)

*Note: This project does not use LLM/AI. Track operational and integration costs instead.*

### Development & Testing Costs

- Vendor API stub calls (if simulating rate limits or external billing)
- Local compute/storage for test data and images
- Settlement file generation and storage volume

### Production Cost Projections

| Cost Type | 100 users | 1K users | 10K users | 100K users |
|-----------|-----------|----------|-----------|------------|
| Vendor API (per check) | $50/mo | $500/mo | $5K/mo | $50K/mo |
| Storage (images + ledger) | $5/mo | $50/mo | $500/mo | $5K/mo |
| Compute (processing) | $20/mo | $200/mo | $2K/mo | $20K/mo |

**Include assumptions:**

- Vendor API charges $0.10 per check validation
- Average 2 checks per user per month
- Image storage: 2 MB per deposit, 90-day retention

---

## Technical Stack

| Layer | Technology |
|------|------------|
| Backend | Go, Java |
| Frontend | Minimal UI (React, Vue, Svelte) or CLI |
| Vendor Integration | REST client; stub in same process or separate service |
| Database/Storage | SQLite, JSON files, PostgreSQL |
| Deployment | Docker Compose, Railway, Render, Fly.io |

Use whatever stack helps you ship. Complete the Pre-Search process to make informed decisions.

---

## Build Strategy

### Priority Order

1. **Transfer state machine** — Define states, transitions, and persistence; everything else depends on it
2. **Vendor Service stub** — Configurable, deterministic responses for all 7+ scenarios
3. **Funding Service** — Business rules, account resolution, ledger posting
4. **Deposit submission flow** — Endpoint/UI → Vendor → Funding → state updates
5. **Operator review workflow** — Queue, approve/reject, audit logging
6. **Settlement file generation** — X9 ICL or equivalent, EOD cutoff
7. **Return/reversal handling** — Notification processing, fee deduction, state transition
8. **Demo interface & tests** — Exercise all paths, one-command setup

### Critical Guidance

- Start with the state machine schema and valid transitions; invalid transitions must be rejected
- Design the stub so evaluators can trigger any scenario without reading your code
- Use transactions for multi-step writes (ledger + transfer state)
- Redact all logs; no real PII, account numbers, or check images
- Synthetic data only; document your test account conventions in README

---

## Required Documentation

| Section | Content |
|---------|---------|
| Architecture | System diagram, service boundaries, data flow, state machine diagram |
| Decision log | Settlement file format, stub design, state machine rationale |
| Risks & limitations | No compliance or regulatory claims; scope boundaries |

---

## Submission Requirements

**Deadline: Sunday 10:59 PM CT**

| Deliverable | Requirements |
|-------------|--------------|
| GitHub Repository | Public repo with README, docs, tests, stub |
| Demo Video | 3–5 min; show happy path, stub scenarios, operator workflow, return handling |
| Pre-Search Document | Saved AI conversation or distilled output |
| Architecture doc | System diagram, service boundaries, data flow |
| Decision log | Key choices and alternatives |
| Operational Cost Analysis | Dev tracking notes + production projection table |
| Deployed Application | Publicly accessible URL |
| Social Post | Tag @GauntletAI |

---

## Interview Preparation

### Technical Topics

- Why this state machine design? What invalid transitions did you explicitly block?
- How would you scale the operator queue for 10x volume?
- Trade-offs: stub in-process vs. separate service; SQLite vs. PostgreSQL
- How does the settlement file format map to real X9 ICL?
- How would you add idempotency for deposit submission and return processing?

### Mindset & Growth

- What would you do differently with a second week?
- How did you prioritize when time was tight?
- What was the hardest bug to track down?
- What did you learn about financial system integration patterns?

---

## Final Note

A simple deposit pipeline with correct state transitions and full audit trail beats a complex one with broken gating. Project completion is required for Austin admission.

---

## Appendix: Pre-Search Checklist

Complete this before writing code. Save your AI conversation as a reference document.

### Phase 1: Define Your Constraints

1. **Scale & Load**
   - How many deposits per day do you need to support in MVP?
   - What is the expected peak submission rate (deposits/minute)?
   - How long must audit logs and check images be retained?

2. **Budget & Resources**
   - What is your time budget for each subsystem (stub, funding, operator UI, settlement)?
   - Are you using any paid services (Vendor API simulation, hosting)?
   - What free-tier limits apply to your chosen stack?

3. **Timeline**
   - What is the latest you can lock the state machine design?
   - When must the stub be stable enough for integration testing?
   - What is the minimum viable operator workflow for MVP?

4. **Compliance & Data Sensitivity**
   - What PII or financial data will you handle? How will you avoid storing real data?
   - What audit requirements apply to operator actions?
   - How will you handle secrets (API keys, account mappings)?

5. **Team & Skills**
   - What is your strongest language (Go vs. Java)? What will you use?
   - Have you worked with state machines or financial ledgers before?
   - What is your preferred local dev setup (Docker, native)?

### Phase 2: Architecture Discovery

1. **Vendor Service Stub**
   - How will you make the stub configurable without code changes?
   - Should the stub run in-process or as a separate service?
   - What request/response schema will you use for IQA, MICR, OCR results?

2. **Funding Service & Ledger**
   - How will you model the omnibus account lookup?
   - What ledger schema supports MOVEMENT entries with To/From/SubType?
   - How will you enforce business rules (limits, duplicates) atomically?

3. **Transfer State Machine**
   - What persistence layer for transfer records?
   - How will you prevent invalid state transitions?
   - How do you handle concurrent updates (e.g., return during settlement)?

4. **Settlement File**
   - What is the minimal X9 ICL structure you need to produce?
   - How will you batch deposits for EOD submission?
   - How do you simulate or stub Settlement Bank acknowledgment?

5. **Operator Workflow**
   - What data must the review queue display for each flagged deposit?
   - How will you store and serve check images (blob storage, base64, file path)?
   - What search/filter indexes do you need?

6. **Return/Reversal**
   - How will return notifications be simulated or received?
   - What ledger entries are required for reversal (debit investor, credit omnibus, fee)?
   - How do you ensure idempotency for duplicate return notifications?

### Phase 3: Post-Stack Refinement

1. **Security & Failure Modes**
   - What happens if the Vendor stub times out mid-validation?
   - How do you handle partial failures (ledger posted but settlement file failed)?
   - What authentication/authorization for operator actions?

2. **Testing**
   - What test framework and structure for unit vs. integration tests?
   - How will you exercise all 7+ stub scenarios in CI?
   - What fixtures or factories for synthetic check data?

3. **Tooling**
   - One-command setup: Makefile, Docker Compose, or both?
   - How will you run the demo script to exercise all paths?
   - What linting/formatting for your chosen language?

4. **Deployment**
   - Where will you deploy (Railway, Render, Fly.io, other)?
   - How will you provide a public URL for evaluation?
   - What environment variables are required? Document in `.env.example`.

5. **Observability**
   - What logging format for deposit decision traces?
   - How will you differentiate deposit sources in logs?
   - What would you add for production monitoring (alerts, metrics)?
