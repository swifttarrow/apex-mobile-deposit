# Task 001: Demo Script

## Goal

Create deterministic demo script that exercises all paths: happy, IQA fail, MICR fail, duplicate, amount mismatch, over-limit, return.

## Deliverables

- [ ] `scripts/demo.sh` or `cmd/demo/main.go`
- [ ] Exercises: happy path, IQA blur, IQA glare, MICR fail → operator approve, duplicate, amount mismatch, over-limit, return/reversal
- [ ] Stub configurability: different inputs → different responses
- [ ] Runnable via `make demo` or documented in README

## Notes

- All 10 testing scenarios from PRD

## Verification

- Demo script runs; all paths exercised
