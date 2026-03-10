# Task 003: Vendor Stub Handler

## Goal

Implement the stub handler that loads config, resolves scenario from account/header, and returns JSON.

## Deliverables

- [ ] `internal/vendor/stub.go` — handler logic
- [ ] Resolve scenario: account prefix from config, then `X-Test-Scenario` header override
- [ ] Return correct JSON for each scenario (7+)

## Notes

- Header `X-Test-Scenario` overrides account-based selection
- Each scenario returns deterministic JSON per PRD spec

## Verification

```bash
go test ./internal/vendor/...
```
