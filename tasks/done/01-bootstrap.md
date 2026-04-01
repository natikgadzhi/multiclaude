# Task 01: Bootstrap Project

## Objective
Set up project scaffolding: main.go, Cobra root command with cli-kit flags, stub commands, Makefile, .goreleaser.yml, CI workflow.

## Acceptance Criteria
- `main.go` calls `cmd.Execute()`
- `cmd/root.go`: Cobra root command with `output.RegisterFlag`, `debug.RegisterFlag`, `config.RegisterFlag("multiclaude")`, version wiring via cli-kit
- `cmd/add.go`, `cmd/use.go`, `cmd/list.go`, `cmd/current.go`, `cmd/remove.go`, `cmd/rename.go`, `cmd/doctor.go`: stub commands that print "not implemented"
- `cmd/backup.go`: stub subcommand group with `create`, `list`, `restore`
- `Makefile` with build, test, lint, vet, ci, install targets + ldflags
- `.goreleaser.yml` for binary releases
- `.github/workflows/ci.yml` running vet + test on push/PR
- `.gitignore` for Go
- `go build ./...` and `go vet ./...` pass
- Error handling uses `errors.ExitWithError(err)` in root.go Execute()
