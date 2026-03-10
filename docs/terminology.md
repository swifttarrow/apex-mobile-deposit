# Terminology

Definitions of tech and domain terms referenced in the Checkstream PRD.

---

## Domain: Check Deposit & Banking

| Term | Definition |
|------|------------|
| **Mobile check deposit** | A feature allowing users to photograph checks and deposit them into accounts via mobile app, eliminating physical check handling. |
| **Check image** | Digital photograph of a physical check (front and/or back) used for deposit processing. |
| **Brokerage platform** | A financial platform (e.g., Schwab, Fidelity) where investors hold accounts and can deposit funds. |
| **Bounced check** | A check that is returned unpaid by the paying bank (e.g., insufficient funds). |
| **Settlement** | The process of finalizing a deposit with the bank, typically via a settlement file sent at end of day. |
| **Settlement Bank** | The bank that receives the settlement file and processes the actual funds transfer. |
| **EOD (End of Day)** | End of business day; for this project, 6:30 PM Central Time (CT) cutoff for same-day settlement. |
| **EOD cutoff** | The deadline (6:30 PM CT) after which deposits roll to the next business day for settlement. |
| **Omnibus account** | A master account that aggregates funds; in this context, the From account for ledger movements (investor funds flow from omnibus to investor account). |
| **Funding** | The process of making deposited funds available to the investor account. |
| **Funding middleware** | Service layer that enforces business rules (limits, duplicates) and routes validated deposits to the ledger. |
| **Contribution type** | Classification of how funds are being added (e.g., deposit type defaults for the account). |

---

## Domain: Validation & Image Processing

| Term | Definition |
|------|------------|
| **IQA (Image Quality Assessment)** | Validation that a check image meets quality standards (focus, lighting). Fails for blur or glare. |
| **IQA pass** | Image quality is acceptable; processing proceeds to MICR/OCR. |
| **IQA fail (blur)** | Image too blurry; user must retake. |
| **IQA fail (glare)** | Glare detected on image; user must retake. |
| **MICR (Magnetic Ink Character Recognition)** | Technology for reading the magnetic ink characters at the bottom of checks (routing number, account number, check number). |
| **MICR read failure** | Vendor could not reliably extract MICR data; deposit is flagged for operator review. |
| **OCR (Optical Character Recognition)** | Technology for reading printed text from images (e.g., written amount on check). |
| **Duplicate detection** | Identifying when the same check has already been deposited; such deposits are rejected. |
| **Amount mismatch** | The amount extracted by OCR differs from the amount entered by the user; flagged for operator review. |
| **Clean pass** | All validations pass; MICR data and amount extracted; deposit proceeds without operator review. |

---

## Domain: Ledger & Transfers

| Term | Definition |
|------|------------|
| **Ledger** | System of record for account balances and financial movements. |
| **Ledger posting** | Creating a record in the ledger that reflects a transfer of funds. |
| **Transfer** | A deposit transaction moving through the pipeline; has a state and lifecycle. |
| **Transfer state machine** | The set of states and allowed transitions for a deposit from submission to completion or rejection. |
| **MOVEMENT** | Ledger entry type representing a funds transfer between accounts. |
| **To AccountId** | The destination account (investor) in a ledger movement. |
| **From AccountId** | The source account (omnibus) in a ledger movement. |
| **SubType DEPOSIT** | Classification of the movement as a deposit. |
| **Transfer Type CHECK** | Classification of the transfer as a check deposit. |
| **Memo FREE** | Memo field value for the ledger entry (as specified in PRD). |
| **Currency USD** | United States Dollar. |
| **Return** | A deposited check that bounces; funds must be reversed. |
| **Reversal** | Undoing a completed deposit; debits the investor and returns funds. |
| **Reversal posting** | Ledger entries that reverse the original deposit and apply any fees. |
| **Returned** | Transfer state indicating the check was returned (bounced). |

---

## Transfer States

| State | Definition |
|-------|------------|
| **Requested** | Deposit submitted; not yet sent to Vendor. |
| **Validating** | Sent to Vendor; awaiting IQA, MICR, OCR, duplicate check. |
| **Analyzing** | Validation passed; business rules and operator review (if flagged) in progress. |
| **Approved** | All checks passed; ready for ledger posting. |
| **FundsPosted** | Ledger posting created; awaiting settlement confirmation. |
| **Completed** | Settlement confirmed; deposit fully processed. |
| **Rejected** | Deposit rejected (IQA fail, duplicate, business rule fail, operator reject). |
| **Returned** | Check bounced; reversal posted; transfer closed. |

---

## Domain: Operator & Compliance

| Term | Definition |
|------|------------|
| **Operator** | Human reviewer who approves or rejects flagged deposits. |
| **Operator review queue** | List of deposits requiring manual review (e.g., MICR failure, amount mismatch). |
| **Flagged deposit** | A deposit that did not auto-pass (e.g., MICR fail, amount mismatch); requires operator action. |
| **Risk score** | Metric indicating likelihood of fraud or error; used in operator queue. |
| **Audit trail** | Log of who did what and when; required for operator actions. |
| **PII (Personally Identifiable Information)** | Data that can identify a person (name, SSN, account numbers); must be redacted in logs. |

---

## Domain: Settlement File

| Term | Definition |
|------|------------|
| **Settlement file** | File sent to the bank at EOD containing deposit data for processing. |
| **X9 ICL** | ANSI X9 standard format for Image Cash Letter—the industry format for electronic check presentment to banks. |
| **Batch metadata** | Information about the settlement batch (date, count, totals). |

---

## Technical Terms

| Term | Definition |
|------|------------|
| **Vendor Service** | External API that performs IQA, MICR, OCR, and duplicate detection on check images. |
| **Vendor API stub** | Simulated Vendor Service used for development and testing; returns configurable responses. |
| **REST client** | HTTP client for calling REST APIs (e.g., Vendor Service). |
| **Idempotency** | Property where repeating the same operation produces the same result; important for deposit submission and return processing to avoid double-posting. |
| **Transaction** | Database unit of work; multi-step writes (e.g., ledger + transfer state) should use transactions for atomicity. |

---

## Project & Process Terms

| Term | Definition |
|------|------------|
| **Pre-Search** | Research phase (1–2 hours) before coding; defines constraints, architecture, and stack decisions. |
| **MVP (Minimum Viable Product)** | Hard gate; all MVP items must be complete to pass (24-hour target). |
| **Happy path** | Ideal flow: submit → validate → approve → ledger posted → settlement → Completed. |
| **Non-happy path** | Rejection or return flows (e.g., IQA fail, duplicate, bounced check). |
| **Test account** | Synthetic account identifier used to trigger specific stub scenarios (e.g., `ACC-IQA-BLUR`). |
| **Synthetic data** | Fake data for testing; no real PII, account numbers, or check images. |

---

## Stack & Deployment (from PRD)

| Term | Definition |
|------|------------|
| **Docker Compose** | Tool for defining and running multi-container Docker applications. |
| **SQLite** | Lightweight file-based SQL database. |
| **PostgreSQL** | Relational database; alternative to SQLite for production use. |
| **Railway, Render, Fly.io** | Cloud platforms for deploying applications. |
| **Go, Java** | Backend language options. |
| **React, Vue, Svelte** | Frontend framework options for minimal UI. |
| **CLI** | Command-line interface; alternative to web UI for deposit submission. |
