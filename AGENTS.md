# Repository Guidelines

## Project Structure & Module Organization
The project follows a standard Go layout. Entrypoints live in `cmd/`, with `cmd/kerja/main.go` wiring dependencies and configuration. Reusable services and domain logic belong in `internal/`; group code by domain (for example, `internal/task` or `internal/http`). Shared helpers that may be consumed externally should be added cautiously under `pkg/`. Tests sit next to the code they cover as `<file>_test.go`, and fixtures or seed data go in `testdata/`.

## Build, Test, and Development Commands
- `go run ./cmd/kerja` — run the service locally using the default configuration.
- `go build ./cmd/kerja` — produce a binary in `./cmd/kerja/kerja`.
- `go test ./...` — execute the full unit-test suite across all packages.
- `go fmt ./... && go vet ./...` — format code and catch common issues before review.

## Coding Style & Naming Conventions
Rely on `gofmt`/`goimports` to enforce tabs for indentation and canonical spacing. Keep exported identifiers CamelCase and meaningful; unexported helpers should remain lowerCamelCase. File names use snake_case (`job_service.go`). Prefer small, focused packages arranged by business capability rather than technical layer. When introducing new dependencies, update `go.mod` and document reasoning in the PR.

## Testing Guidelines
Write table-driven tests whenever feasible and keep assertions minimal but descriptive. Target stable coverage for core packages by running `go test -cover ./...`; flag drops in coverage in your PR description. Integration or HTTP handler tests should live under `internal/<domain>/integration_test.go` and may leverage sample payloads from `testdata/`. Ensure any new functionality includes failure-path coverage.

## Commit & Pull Request Guidelines
Commits should stay small, in the imperative mood (“Add task scheduler”). Use the body to capture rationale or migration notes, mirroring the concise history already in use. Each PR must include: a summary of changes, linked issue or task ID, testing evidence (`go test ./...` output is sufficient), and screenshots or curl transcripts for observable endpoints. Request review once CI is green and respond promptly to feedback.

## Release & Distribution
- Releases are driven by GoReleaser using `.goreleaser.yaml`; it cross-compiles macOS (arm64/amd64) and Linux binaries, emits archives + checksums, and updates the Homebrew formula.
- Tag the repository with `vX.Y.Z` to trigger the `release` GitHub Actions workflow. It runs tests, invokes GoReleaser, and publishes GitHub Releases.
- Provide a personal access token with push rights to `faizmokh/homebrew-tap` as the `HOMEBREW_TAP_GITHUB_TOKEN` secret so the formula bump succeeds.
- After a release, smoke-test the tap locally (`brew tap faizmokh/tap && brew install kerja`) and confirm `kerja --version` reports the new tag + commit.
