# kerja

kerja is a Go-based terminal application (Bubble Tea TUI + Cobra CLI) for capturing and reviewing daily work logs persisted as Markdown. Storage behavior is defined in `SPEC.md`.

---

## Installation

```bash
go build ./cmd/kerja
./cmd/kerja/kerja --help
```

By default the app stores Markdown under `~/.kerja/YYYY/YYYY-MM.md`. Override the root with `KERJA_HOME` (for example `export KERJA_HOME=~/worklogs`).

---

## CLI Overview

The CLI mirrors the TUI data model so commands can script the same workflows the UI exposes.

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

All timestamps are interpreted in the current locale unless otherwise noted. For search, prefix a term with `#` to match tags exactly; add `--include-text` to also scan entry bodies when using tag-prefixed queries. `--json` emits results suitable for piping into other tools.

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

Running `kerja` with no subcommand boots the Bubble Tea interface. The model loads today's section and gives you quick access to nearby days and entry actions.

- `h`/left or `l`/right switch between the previous and next day
- `t` jumps back to today, `r` refreshes the current section
- `j`/down and `k`/up change the focused entry
- Space or `x` toggles the focused entry between todo and done
- `a` appends a todo entry, `A` appends a done entry (text then optional `#tags`)
- `e` edits the focused entry’s text/tags, `T` updates its time, `S` updates status, `d` removes it (press `y` to confirm)
- `Esc` cancels any in-progress dialog
- `q` or `Ctrl+C` exits the program

Entry prompts accept the same tokens as the CLI helpers: add `@HH:MM` to set the timestamp, `!todo`/`!done` to choose status, and `#tag` for labels.

Sections that do not exist yet render as `(no entries)` so you can see what still needs logging. The TUI shares the same reader and writer as the CLI, so changes are written to the Markdown log immediately.

---

## Development

| Task | Command |
|------|---------|
| Run tests | `go test ./...` |
| Lint/format | `go fmt ./... && go vet ./...` |
| TUI dev loop | `go run ./cmd/kerja` |

`internal/cli/integration_test.go` exercises the CLI end-to-end (append → list → search → edit → delete) against temp logbooks so regressions surface early. The codebase follows conventional Go layouts (`cmd/`, `internal/`); see `SPEC.md` for the Markdown schema and write rules.
