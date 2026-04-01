# Task 19: Homebrew Formula and Release Workflow

## Objective
Set up GoReleaser and Homebrew tap for easy installation.

## Acceptance Criteria
- `.goreleaser.yml` builds for darwin/linux amd64/arm64
- `.github/workflows/release.yml` with `workflow_dispatch` version bump (matches other tools)
- Homebrew formula in `github.com/natikgadzhi/taps` repo
- `brew install natikgadzhi/taps/multiclaude` works
- `multiclaude version` shows correct version, commit, date

## Dependencies
- Task 18 (README polish)
