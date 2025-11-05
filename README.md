# kerja

kerja is a Go-based terminal companion for capturing, reviewing, and searching daily work logs. It ships both a Bubble Tea TUI and a Cobra-powered CLI, storing every entry as plain Markdown so your notes stay portable and version-control friendly.

## Features
- Capture todo/done entries from a keyboard-driven TUI or scriptable CLI.
- Persist monthly logs as Markdown that diff cleanly and are easy to sync.
- Jump across days, list rolling windows, and search by keyword or tag.
- Share a single storage engine between the CLI and TUI to keep workflows unified.
- Ship cross-platform binaries via Homebrew or `go build`.

## Requirements
- Go 1.25 or newer when building from source.
- macOS or Linux terminal; Windows works via WSL.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap faizmokh/tap
brew install kerja
```

### From Source

```bash
go install github.com/faizmokh/kerja/cmd/kerja@latest
kerja --help
```

To build locally without touching `GOBIN`:

```bash
go build ./cmd/kerja
./cmd/kerja/kerja --help
```

Check metadata with `kerja --version`.

## Quick Start

```bash
kerja todo "Write integration tests" #testing
kerja log --time 09:15 "Review design doc" #review
kerja list --week
```

The default log location is `~/.kerja/<year>/<year-month>.md`. Set `KERJA_HOME` to point at a different root (for example `export KERJA_HOME=~/worklogs`).

Launch the TUI by running `kerja` with no arguments. It opens today's section and keeps the file in sync as you add, edit, toggle, or delete entries.

## CLI Commands

The CLI mirrors the TUI data model so you can automate the same workflows.

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `kerja today` | Print entries for today (or `--date`) | `--date=YYYY-MM-DD` |
| `kerja prev` / `kerja next` | Navigate relative to a date | `--date=YYYY-MM-DD` |
| `kerja jump <date>` | Jump directly to a specific day | `YYYY-MM-DD` |
| `kerja list` | List entries over a rolling window | `--date` (default today), `--days`, `--week` |
| `kerja search <term>` | Search current month by text or tag | `--date`, `--case-sensitive`, `--include-text`, `--json` |
| `kerja log [text ... #tags]` | Append a done entry | `--date`, `--time` |
| `kerja todo [text ... #tags]` | Append a todo entry | `--date`, `--time` |
| `kerja toggle <index>` | Flip todo/done status | `--date` |
| `kerja edit <index> [text ... #tags]` | Update text/tags/time/status | `--date`, `--time`, `--status` |
| `kerja delete <index>` | Remove an entry | `--date` |

Timestamps use your local timezone. For search, prefix a term with `#` to match tags exactly; add `--include-text` to also scan entry bodies. `--json` emits results you can pipe into other tools.

## Example Workflow

```bash
kerja todo "Write integration tests" #testing
kerja log --time 09:15 "Review design doc" #review
kerja list --week
kerja search bug
kerja toggle 1
kerja edit 2 "Review design doc updates" --status done
kerja delete 3
```

This sequence appends todo + done entries, lists the past week, searches for "bug", toggles completion, edits an entry (including status change), and finally deletes an entry.

## TUI

Running `kerja` with no subcommand boots the Bubble Tea interface. The model loads today's section and gives you quick access to nearby days and entry actions.

- `h`/left or `l`/right switch between the previous and next day
- `t` jumps back to today, `r` refreshes the current section
- `j`/down and `k`/up change the focused entry
- Space or `x` toggles the focused entry between todo and done
- `a` appends a todo entry, `A` appends a done entry (text then optional `#tags`)
- `e` edits the focused entry’s text/tags, `T` updates its time, `S` updates status, `d` removes it (press `y` to confirm)
- `Esc` cancels any in-progress dialog
- `q` or `Ctrl+C` exits the program

Entry prompts accept the same tokens as the CLI helpers: add `@HH:MM` to set the timestamp, `!todo`/`!done` to choose status, and `#tag` for labels. Sections that do not exist yet render as `(no entries)` so you can see what still needs logging. The TUI shares the same reader and writer as the CLI, so changes are written to the Markdown log immediately.

## Data & Storage Format

- Logs live under `~/.kerja/` by default, grouped `/year/year-month.md`.
- Each file contains a `# {Month Name} {Year}` heading and daily `## YYYY-MM-DD` sections.
- Entries take the form `- [ ] [HH:MM] Task text #tag1 #tag2` (`[x]` marks done).
- Parser and writer rules are documented in `SPEC.md`; refer there for edge cases and write guarantees.

This structure keeps files human-friendly while enabling reliable parsing for both the CLI and TUI layers.

## Project Layout

- `cmd/kerja`: application entrypoint wiring Cobra/TUI bootstrap.
- `internal/cli`: command implementations and integration tests.
- `internal/files`: filesystem helpers, including `KERJA_HOME` overrides.
- `internal/logbook`: Markdown parser, reader, and writer.
- `internal/ui`: Bubble Tea models for the interactive interface.
- `internal/version`: runtime version metadata surfaced via `kerja --version`.

## Development

| Task | Command |
|------|---------|
| Run tests | `go test ./...` |
| Lint/format | `go fmt ./... && go vet ./...` |
| TUI dev loop | `go run ./cmd/kerja` |

`internal/cli/integration_test.go` exercises the CLI end-to-end (append → list → search → edit → delete) against temporary logbooks so regressions surface early. Run `go test -cover ./...` locally to keep coverage steady.

## Release Process

1. Ensure the main branch is green and `go test ./...` passes locally.
2. Export a GitHub token with permission to push to `faizmokh/homebrew-tap` (configure it as `HOMEBREW_TAP_GITHUB_TOKEN` in repository secrets).
3. Tag the commit using semantic versioning (for example `git tag v0.4.0 && git push origin v0.4.0`).
4. GitHub Actions runs GoReleaser to create archives, checksums, and update the Homebrew tap formula.
5. Verify the release by installing from Homebrew (`brew tap faizmokh/tap && brew install kerja`) and running `kerja --version`.
