# Tests

Unit and integration tests live alongside source code in `internal/*_test.go`. This folder satisfies the deliverable structure; run instructions are below.

## Running tests

From the project root:

```bash
make test
```

Or with verbose output:

```bash
go test -v ./...
```

## Test report (deliverable)

Generate the scenario coverage report under `reports/`:

```bash
make test-report
```

This writes `reports/test_report.txt` with full test output.

## Coverage

Tests cover:

- **Happy path** — end-to-end deposit → validation → approval → ledger → settlement
- **Vendor stub scenarios** — IQA blur/glare, MICR fail, duplicate, amount mismatch, clean pass
- **Business rules** — deposit limits, contribution defaults, eligibility, duplicate detection
- **State machine** — valid and invalid transitions (including Completed → Returned)
- **Reversal** — return processing with $30 fee, ledger reversal entries
- **Settlement** — file generation, EOD cutoff, batch contents
- **Operator workflow** — queue, approve, reject, audit

See `internal/api/scenarios_test.go` for scenario-driven tests that exercise all paths.
