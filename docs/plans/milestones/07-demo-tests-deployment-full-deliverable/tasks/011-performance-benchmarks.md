# Task 011: Performance Benchmarks

## Goal

Document and validate performance expectations per [requirements.md](../../../requirements.md) § Performance Benchmarks.

## Deliverables

- [ ] **Document targets** in README or docs/architecture.md:
  - Validation round-trip: Vendor Service stub responds < 1 second
  - Ledger posting: Transfer created and posted within seconds of approval
  - Settlement file generation: Batch file produced within seconds of EOD trigger
  - Operator queue: Flagged items surface in review queue immediately upon flagging
  - State transitions: All state changes propagate and are queryable within 1 second
- [ ] **Benchmark or smoke test** (optional): `make bench` or integration test that asserts validation round-trip < 1s
- [ ] **Decision log:** Document any performance trade-offs or simplifications for MVP

## Notes

- For stub, validation round-trip should trivially meet < 1s; test confirms no regression
- Settlement "within seconds" is relative to trigger; document expected batch size

## Verification

- [ ] Performance targets documented
- [ ] At least one benchmark or assertion (e.g. validation latency) in test suite
- [ ] `make test` passes including performance-related checks
