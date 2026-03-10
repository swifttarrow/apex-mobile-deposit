# Task 001: Go Module & Makefile

## Goal

Create the Go module and Makefile so the project can be built, tested, and run with one command.

## Deliverables

- [ ] `go.mod` with module path (e.g. `github.com/.../checkstream`)
- [ ] `Makefile` with targets: `make dev`, `make test`, `make build`
- [ ] `go build ./...` succeeds (even if no code yet)

## Notes

- Use `go mod init` for module creation
- `make dev` should run the server (e.g. `go run ./cmd/server`)
- `make test` should run `go test ./...`

## Verification

```bash
go build ./...
make test
make dev  # Ctrl+C to stop
```
