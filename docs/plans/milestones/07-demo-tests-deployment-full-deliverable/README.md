# Milestone 7: Demo, Tests, One-Command Setup & Full Deliverable

## Overview

Minimal UI or CLI for demo; tests covering all 10 scenarios; `make dev`; README; deploy to Railway/Render/Fly.io. Full deliverable: risks doc, operational cost analysis, architecture with state machine diagram, SUBMISSION.md, test report, demo video, Pre-Search reference, social post, ledger view.

**Source:** [MVP Plan Phase 7](../../2025-03-10-checkstream-mvp.md#phase-7-demo-tests-one-command-setup--deployment) | [Gaps: Documentation, Submission](../../2025-03-10-checkstream-full-deliverable.md#8-documentation-gaps)

## Dependencies

- [ ] Milestones 1–6 complete

## Changes Required

- Makefile: `make dev`, `make test`, `make test-report`
- docker-compose.yml (optional)
- Demo script (scripts/demo.sh or cmd/demo/main.go)
- README, docs/architecture.md, docs/decision_log.md, docs/risks_limitations.md, docs/operational_cost.md
- State machine diagram in architecture
- SUBMISSION.md template
- .env.example
- Integration tests for all 10 scenarios
- Ledger view: GET /ledger or /accounts/:id/balance
- Observability: decision trace, deposit source tags, settlement monitoring, redacted logs
- Performance benchmarks documented and validated
- Test report to reports/
- Demo video (3–5 min)
- Pre-Search reference in README
- Social post checklist

## Success Criteria

### Automated Verification

- [ ] `make test` — all tests pass (including performance assertions)
- [ ] `make dev` — server starts; demo script runs
- [ ] `make test-report` — output to reports/

### Manual Verification

- [ ] README instructions work for fresh clone
- [ ] Deployed app publicly accessible
- [ ] Demo script exercises all 10 scenarios
- [ ] Demo video recorded (3–5 min)
- [ ] Pre-Search document referenced

## Tasks

- [001-demo-script](./tasks/001-demo-script.md)
- [002-ten-scenario-tests](./tasks/002-ten-scenario-tests.md)
- [003-readme-setup](./tasks/003-readme-setup.md)
- [004-architecture-decision-log](./tasks/004-architecture-decision-log.md)
- [005-risks-operational-cost](./tasks/005-risks-operational-cost.md)
- [006-submission-test-report](./tasks/006-submission-test-report.md)
- [007-ledger-view](./tasks/007-ledger-view.md)
- [008-docker-deploy](./tasks/008-docker-deploy.md)
- [009-demo-video-final-checklist](./tasks/009-demo-video-final-checklist.md)
- [010-observability-monitoring](./tasks/010-observability-monitoring.md)
- [011-performance-benchmarks](./tasks/011-performance-benchmarks.md)
