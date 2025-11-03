# Build Plan

## 1. Foundations
1. Initialize Go module `github.com/faizmokh/kerja`; configure Go 1.22+.  
2. Sketch `cmd/kerja` entrypoint that wires Cobra CLI + Bubble Tea program.  
3. Define shared config/env loader so both CLI and TUI resolve paths under `~/.kerja/`.  
4. Set up scaffolding packages: `internal/files`, `internal/logbook`, `internal/ui`, `internal/cli`.

## 2. Markdown Storage Layer
5. Implement `internal/files.Manager` for month file discovery, creation, and atomic writes.  
6. Build `internal/logbook.Parser` to stream entries per the SPEC schema, tolerating malformed lines.  
7. Create `internal/logbook.Writer` supporting append, toggle, edit, delete, with table-driven tests for each operation.  
8. Add integration tests using temp dirs to assert Manager + Writer generate correct Markdown.

## 3. CLI Workflow
9. Wire Cobra commands: `today`, `prev`, `next`, `list`, `search`, `jump`, `log`, `todo`, `toggle`, `edit`, `delete`.  
10. Share a command context that resolves target date, loads entries, applies Writer operations, and prints results.  
11. Provide rich help text and examples matching SPEC verbs; surface JSON output flag for scripting.  
12. Cover command behaviors with unit tests (e.g., using golden Markdown fixtures in `testdata/cli`).

## 4. TUI Experience
13. Model Bubble Tea state: current date, entries, focused index, modal state.  
14. Implement panes for entry list, detail preview, and command palette; ensure keyboard shortcuts for navigation (prev/next day, toggle, edit).  
15. Connect TUI actions to logbook Writer via async commands; handle optimistic updates and error toasts.  
16. Add theming (lipgloss) plus responsive layout tests via `bubbles/viewport`.

## 5. Synchronization & Extensibility
17. Introduce background watcher to reload when Markdown edited externally.  
18. Add optional YAML front matter parsing hook while maintaining backward compatibility.  
19. Provide export subcommand (`kerja export --format=json|csv`) leveraging parser outputs.

## 6. Packaging & Distribution
20. Integrate CI (GitHub Actions) running `go test ./...`, linters (`golangci-lint`).  
21. Build release pipeline producing binaries for macOS/Linux and generating checksums.  
22. Publish Homebrew tap formula with version bump automation.  
23. Document installation, commands, and contribution flow in README/AGENTS.
