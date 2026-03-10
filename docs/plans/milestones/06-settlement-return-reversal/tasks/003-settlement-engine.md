# Task 003: Settlement Engine

## Goal

Implement settlement engine: batch FundsPosted by EOD date, generate X9 ICL with MICR, images, amounts.

## Deliverables

- [ ] `internal/settlement/engine.go` — batch query, X9 build
- [ ] Query transfers in FundsPosted for settlement date (respect EOD cutoff)
- [ ] Build X9: File Header, Cash Letter, Bundle, Check Detail, Image View, Controls
- [ ] Write file to disk (e.g. `./settlement/YYYYMMDD.x9`)
- [ ] Transition FundsPosted → Completed
- [ ] Assert MICR, images, amounts present in generated file (test)

## Notes

- Use moov-io/imagecashletter for X9 structure

## Verification

```bash
go test ./internal/settlement/...
```
