# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**kerja** is a Go-based terminal application for daily work logging with both TUI (Bubble Tea) and CLI (Cobra) interfaces. It stores entries as human-readable Markdown files in `~/.kerja/` by default, organized by month.

## Development Commands

### Building and Running
```bash
# Build locally
go build ./cmd/kerja
./cmd/kerja/kerja --help

# Install from source
go install github.com/faizmokh/kerja/cmd/kerja@latest

# Run TUI development mode
go run ./cmd/kerja
```

### Testing and Quality
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Lint and format
go fmt ./... && go vet ./...
```

### Release Process
```bash
# Tag and push (triggers GitHub Actions + GoReleaser)
git tag v0.4.0
git push origin v0.4.0
```

## Architecture Overview

### Core Components
- **cmd/kerja/main.go**: Entry point that delegates to CLI package
- **internal/cli/**: All CLI command implementations (today, list, search, log, todo, toggle, edit, delete)
- **internal/ui/model.go**: Bubble Tea TUI implementation with keyboard navigation
- **internal/logbook/**: Markdown parsing, reading, and writing of log entries
- **internal/files/**: File system management and `KERJA_HOME` path resolution
- **internal/version/**: Build-time version metadata

### Data Storage Format
- **Location**: `~/.kerja/<year>/<year-month>.md` (configurable via `KERJA_HOME`)
- **Structure**: Monthly Markdown files with `# Month Year` headings and `## YYYY-MM-DD` sections
- **Entry Format**: `- [x] [HH:MM] Task text #tag1 #tag2` (done) or `- [ ] [HH:MM] Task text #tag1 #tag2` (todo)

### Key Interfaces
The application shares the same data layer between CLI and TUI:
- **CLI**: Scriptable commands for automation
- **TUI**: Interactive keyboard-driven interface (launch with `kerja` with no args)

## Essential CLI Commands
```bash
# Core workflow
kerja todo "Write integration tests" #testing
kerja log "Review design doc" #review
kerja today
kerja list --week
kerja search bug

# Entry management
kerja toggle <index>
kerja edit <index> "Updated text" #newtag
kerja delete <index>
```

## TUI Controls
- `h`/`l`: Previous/next day
- `j`/`k`: Navigate entries
- Space/`x`: Toggle todo/done
- `a`/`A`: Add todo/done entry
- `e`: Edit entry, `d`: Delete entry
- `t`: Back to today, `r`: Refresh

## Testing Strategy

### Test Structure
- **Unit tests**: Individual component testing in `internal/logbook/`, `internal/cli/`, `internal/files/`
- **Integration tests**: `internal/cli/integration_test.go` for end-to-end CLI workflows
- **Mock file system**: Tests use temporary directories for isolation

### Key Test Areas
- Markdown parser reliability (SPEC.md compliance)
- CLI command argument parsing and execution
- File operations (atomic writes, path resolution)
- Entry CRUD operations

## Critical Implementation Details

### Markdown Schema (SPEC.md)
The application strictly follows the specification in `SPEC.md`:
- Entry regex: `^- \[( |x)\] \[(\d{2}:\d{2})\] (.*?)(?:\s(#\w+))*\s*$`
- Atomic file operations (temp write + rename)
- UTF-8 encoding, max ~5MB per month

### Dual Interface Architecture
Both CLI and TUI share:
- Same storage engine (`internal/logbook/`)
- Same data models and validation
- Same file management (`internal/files/`)

### Release Automation
- **GoReleaser**: Cross-platform builds (darwin/linux, amd64/arm64)
- **Homebrew**: Automatic tap updates via GitHub Actions
- **Version**: Runtime version from `internal/version/`

## Development Environment
- **Go Version**: 1.25.3+
- **Dependencies**: Bubble Tea, Cobra, Charm libraries (lipgloss, gum)
- **Platforms**: macOS, Linux (Windows via WSL)
- **Config**: `KERJA_HOME` environment variable for custom storage path

## File Format Guarantees
- Human-readable and Git-friendly
- Append-first, never reorders entries
- Parser tolerant of incomplete files
- All writes are atomic operations