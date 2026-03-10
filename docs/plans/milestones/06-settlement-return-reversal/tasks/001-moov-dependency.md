# Task 001: Moov Dependency

## Goal

Add moov-io/imagecashletter for X9 ICL generation.

## Deliverables

- [ ] `go.mod` — add `github.com/moov-io/imagecashletter`
- [ ] `go mod tidy` succeeds

## Notes

- X9 ICL format for settlement file

## Verification

```bash
go mod download
go build ./...
```
