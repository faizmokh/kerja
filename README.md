# kerja

kerja is a Go-based terminal application (Bubble Tea TUI + Cobra CLI) for capturing and reviewing daily work logs persisted as Markdown. Storage behavior is defined in `SPEC.md`.

---

## Installation

```bash
go build ./cmd/kerja
./cmd/kerja/kerja --help
```

By default the app stores Markdown under `~/.kerja/YYYY/YYYY-MM.md`. Override the root with `KERJA_HOME` (implementation coming next).

---

## CLI Overview

The CLI mirrors the TUI data model so commands can script the same workflows the UI exposes.

| Command | Purpose | Key Flags |
|---------|---------|-----------|
| `kerja today` | Print entries for today (or `--date`) | `--date=YYYY-MM-DD` |
| `kerja prev` / `kerja next` | Navigate relative to a date | `--date=YYYY-MM-DD` |
| `kerja jump <date>` | Jump directly to a specific day | `YYYY-MM-DD` |
| `kerja list` | List entries over a rolling window | `--date` (default today), `--days`, `--week` |
| `kerja search <term>` | Search current month by text or tag | `--date` (month anchor) |
| `kerja log [text ... #tags]` | Append a done entry | `--date`, `--time` |
| `kerja todo [text ... #tags]` | Append a todo entry | `--date`, `--time` |
| `kerja toggle <index>` | Flip todo/done status | `--date` |
| `kerja edit <index> [text ... #tags]` | Update text/tags/time/status | `--date`, `--time`, `--status` |
| `kerja delete <index>` | Remove an entry | `--date` |

All timestamps are interpreted in the current locale unless otherwise noted.

---

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

This sequence appends todo + done entries, lists the past week, searches for “bug”, toggles completion, edits an entry (including status change), and finally deletes an entry.

---

## TUI

Running `kerja` with no subcommand boots the Bubble Tea interface. UI features (panels, shortcuts, etc.) are under active development and will be documented as they land.

---

## Development

| Task | Command |
|------|---------|
| Run tests | `go test ./...` |
| Lint/format | `go fmt ./... && go vet ./...` |
| TUI dev loop | `go run ./cmd/kerja` |

`internal/cli/integration_test.go` exercises the CLI end-to-end (append → list → search → edit → delete) against temp logbooks so regressions surface early. The codebase follows conventional Go layouts (`cmd/`, `internal/`); see `SPEC.md` for the Markdown schema and write rules.
