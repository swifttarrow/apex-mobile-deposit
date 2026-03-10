# Milestone 2: Vendor Service Stub

## Overview

Config-driven stub returning 7+ differentiated responses. Selection via `scenarios.json` (account prefix) and `X-Test-Scenario` header. Include MICR confidence scores for operator queue (full deliverable).

**Source:** [MVP Plan Phase 2](../../../thoughts/plans/2025-03-10-checkstream-mvp.md#phase-2-vendor-service-stub) | [Gaps: MICR confidence](../../../thoughts/plans/2025-03-10-checkstream-mvp-to-full-deliverable.md#2-operator-workflow-gaps)

## Dependencies

- [ ] Milestone 1: Project Setup & Transfer State Machine

## Changes Required

- `config/scenarios.json` — map account prefixes → scenario names
- `internal/vendor/stub.go` — handler, load config, resolve scenario from account/header
- `internal/vendor/types.go` — request/response structs
- `cmd/server/main.go` — register `POST /vendor/validate`
- Vendor responses include `confidence` or `iqScore` where applicable (for operator queue)

## Success Criteria

### Automated Verification

- [ ] `go test ./internal/vendor/...` — each scenario returns expected JSON
- [ ] Different account IDs produce different responses per config

### Manual Verification

- [ ] `curl -X POST .../vendor/validate` with `account_id: ACC-IQA-BLUR` returns blur fail
- [ ] `X-Test-Scenario: clean_pass` overrides account-based selection

## Tasks

- [001-scenarios-config](./tasks/001-scenarios-config.md)
- [002-vendor-types](./tasks/002-vendor-types.md)
- [003-vendor-stub-handler](./tasks/003-vendor-stub-handler.md)
- [004-register-route](./tasks/004-register-route.md)
