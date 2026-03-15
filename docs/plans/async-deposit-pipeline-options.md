# Async Deposit Pipeline: Options

## Current behavior

`POST /deposits` runs the full pipeline in the request handler:

1. Create transfer (Requested) → transition to Validating, persist
2. Call vendor stub (sync)
3. On pass: Validating → Analyzing → business rules → Approved → ledger → **FundsPosted** → return **201** with transfer  
4. On fail/reject: persist Rejected, return 4xx with transfer  
5. On flagged: persist Analyzing, return **202** with transfer  

So a “clean pass” deposit blocks until vendor + funding + ledger complete and returns 201 with state `FundsPosted`. The mobile then sees FundsPosted immediately instead of Submitted → Processing → Funds Posted.

## Goal

- `POST /deposits` should **accept** the deposit and return quickly (e.g. **202 Accepted**) with the transfer in **Requested** or **Validating**.
- The pipeline (vendor → business rules → ledger → state updates) should run **asynchronously**.
- Clients (e.g. mobile) can poll `GET /deposits/:id` to see state move: Requested → Validating → Analyzing → (Approved →) FundsPosted (or Rejected/Returned).

## What the async worker needs

The existing pipeline logic (vendor call, business rules, ledger, transitions) needs, per deposit:

- **Transfer ID** (already created and persisted)
- **Request payload**: `account_id`, `amount`, `front_image`, `back_image`, `scenario`, `source` (and any new fields)

Today the handler only persists transfer rows with `id, account_id, amount, state, created_at, updated_at`. Image paths and scenario are set in memory and used in the same request. So for async we must either:

- **A)** Persist enough on the transfer (e.g. `front_image_path`, `back_image_path`, and a small metadata/scenario field) before returning 202, so the worker can load the transfer by ID and run the pipeline, or  
- **B)** Store the full request (or a job payload) in a separate table or message so the worker has everything without re-reading the request body.

---

## Option 1: In-process goroutine (no queue)

**Idea:**  
`POST /deposits` creates the transfer, persists it (and any needed request fields), starts a **goroutine** that runs the current pipeline (vendor → rules → ledger), and returns **202** with the transfer in Requested (or Validating).

**Pros**

- No new dependencies or infrastructure
- Minimal code change: extract “run pipeline from Validating onward” into a function, call it in a `go func() { ... }()`
- Fits current single-binary, SQLite, single-node setup

**Cons**

- Jobs are **in-memory only**. Process restart loses any in-flight work (transfers stuck in Requested/Validating until something else moves them).
- No backpressure: a burst of deposits spawns many goroutines (vendor/DB can be overloaded). You can add a semaphore or a small in-memory queue with a fixed worker pool to limit concurrency.
- No horizontal scaling of workers (single process).

**Good for:** Local/demo, or as a first step before adding a real queue.

---

## Option 2: Database-backed job table (polling worker)

**Idea:**  
Add a table, e.g. `deposit_jobs (id, transfer_id, status, created_at, updated_at)` (and optionally `payload JSON` or store request fields on the transfer).  
`POST /deposits` creates the transfer, persists request data needed for processing, inserts a row in `deposit_jobs` with status `pending`, returns **202**.  
A **background goroutine** (or separate worker process) **polls** `deposit_jobs` for `pending`, claims a row (e.g. `UPDATE ... SET status = 'running' WHERE id = ? AND status = 'pending'`), runs the pipeline, then sets status to `done` or `failed`.

**Pros**

- No new infrastructure beyond the existing DB (SQLite)
- Survives process restart: pending jobs remain in the table
- Same binary can run the HTTP server and the polling loop (like the existing settlement ticker in `main.go`)
- Optional: store retry count and last_error for failed jobs

**Cons**

- Polling latency (e.g. 1–5 s) unless you poll frequently
- Concurrency: need careful “claim” semantics (e.g. by status + `updated_at`) so only one worker processes a job; with SQLite, single writer is natural
- You still need to persist enough request data (images, scenario) on transfer or in job payload

**Good for:** Demo + “no new services,” with durable jobs and simple deployment.

---

## Option 3: Redis (or Redis-compatible) queue

**Idea:**  
Use a Redis list or a library such as **asynq** (Redis-backed job queue for Go).  
`POST /deposits` creates the transfer, persists minimal data, **enqueues** a job (e.g. `{ "transfer_id": "..." }` or full payload), returns **202**.  
One or more **workers** (same process or separate) **dequeue** jobs and run the pipeline.

**Pros**

- Fast, low-latency processing
- At-least-once delivery if you use ack/retry (e.g. asynq)
- Natural backpressure and worker scaling (N workers)
- Fits multi-instance deployment (e.g. multiple API servers + shared Redis)

**Cons**

- New dependency: Redis (or Upstash, etc.)
- Operational cost and deployment (even if small)

**Good for:** Production or when you want a “real” queue without running a full message broker.

---

## Option 4: Dedicated message broker (SQS, RabbitMQ, NATS, etc.)

**Idea:**  
Same as Option 3, but the queue is AWS SQS, RabbitMQ, NATS Jetstream, etc.  
`POST /deposits` publishes a message (transfer_id + payload or reference); workers consume and run the pipeline.

**Pros**

- Durable, at-least-once (or exactly-once with care), dead-letter queues, scaling
- Standard pattern for production event-driven systems

**Cons**

- More infrastructure and configuration
- Likely overkill for a single-node demo unless you explicitly want to exercise a broker

**Good for:** Production at scale or when the rest of the system already uses a broker.

---

## Recommendation (by context)

| Context | Suggestion |
|--------|------------|
| **Minimal change / demo** | **Option 1 (goroutine)**. Return 202 after creating transfer and persisting needed fields; run pipeline in a goroutine. Optionally add a small in-memory worker pool + channel to cap concurrency. |
| **Durable, no new services** | **Option 2 (DB job table)**. POST creates transfer + job row, returns 202; one or more polling workers run the pipeline. Survives restarts and keeps the “single binary + SQLite” story. |
| **Production / multi-node** | **Option 3 (Redis)** or **Option 4 (broker)**. Use asynq + Redis for a good balance of simplicity and robustness; move to SQS/RabbitMQ/NATS if you already have or want that stack. |

---

## Implementation outline (any option)

1. **Persist request data for async**  
   Ensure the worker can run without the HTTP request body: e.g. persist `front_image_path`, `back_image_path`, and scenario (or a small JSON blob) on the transfer or in a job row when creating the transfer.

2. **Split handler**  
   - **Synchronous part:** Validate request → create transfer (Requested) → persist transfer (+ image paths, scenario) → enqueue or start async work → return **202** with transfer.  
   - **Async part:** “Process deposit” function that loads transfer (and any job payload), runs from Validating (vendor) through FundsPosted/Rejected/Analyzing, with same logic as today.

3. **Response contract**  
   - **202 Accepted:** Body = `{ "message": "deposit accepted", "transfer": { "id": "...", "state": "Requested", ... } }`.  
   - Client uses `GET /deposits/:id` to poll until state is terminal (FundsPosted, Rejected, etc.).

4. **Mobile (and tests)**  
   - Expect 202 for new deposits; poll `GET /deposits/:id` on the status screen until state is terminal (or show stepper that updates on poll).  
   - E2E and API tests: assert 202 and optional polling until FundsPosted for happy path.

5. **Idempotency**  
   Existing idempotency (e.g. `X-Idempotency-Key`) should still apply to the **create + enqueue** step so a duplicate request doesn’t create two transfers or two jobs.

---

## Next step

Choose one of the options above (e.g. start with **Option 1** for speed, or **Option 2** for durability without new deps), then implement: (1) persist request data for async, (2) 202 response path, (3) extracted pipeline function, (4) worker (goroutine, poller, or Redis worker), (5) client polling and tests.
