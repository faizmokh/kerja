kerja – Markdown File Specification
Version: 1.1
Format: Markdown-based structured log
Purpose: Defines the storage schema and behavior for the TUI-based daily tracker.

----------------------------------------
Overview
----------------------------------------
This specification describes how the kerja app stores and manages daily work logs in Markdown format.

Goals:
- Human-readable, Git- and diff-friendly.
- Appendable, editable, and deletable within clear structure.
- Consistent schema for reliable parsing and manipulation.

----------------------------------------
Tech spec
----------------------------------------
- A golang TUI app build using using [bubbletea](https://github.com/charmbracelet/bubbletea)
- It should allow usage through both TUI and cli command
- It should be able to distributed through homebrew

----------------------------------------
1. Storage Layout
----------------------------------------
Directory Structure:
~/.kerja/
  └── 2025/
      ├── 2025-10.md
      ├── 2025-11.md

File Naming:
YYYY-MM.md
Example: 2025-11.md → November 2025 log file
Each file represents one month of activity.

----------------------------------------
2. File Schema
----------------------------------------
Example File:

# November 2025

## 2025-11-02
- [x] [09:45] Fixed loan summary layout #ui #bug
- [ ] [11:10] Review PR for lending dashboard #review
- [x] [14:20] Investigated caching strategy #lending #research

## 2025-11-03
- [x] [10:00] Implemented caching for user profiles #lending #feature
- [ ] [15:30] Write unit tests for caching #testing

----------------------------------------
3. Schema Rules
----------------------------------------
Month Heading:
- Markdown heading level 1 (#)
- Format: {Full Month Name} {Year}
- Used for display only

Date Section:
- Markdown heading level 2 (##)
- Format: YYYY-MM-DD
- Must precede all entries for that date
- Only one per date per file

Entry Line:
- [ ] [HH:MM] Task text #tag1 #tag2 ...
or
- [x] [HH:MM] Task text #tag1 #tag2 ...

Regex:
^- \[( |x)\] \[(\d{2}:\d{2})\] (.*?)(?:\s(#\w+))*\s*$

Fields:
status: enum(todo, done)
time: string (HH:MM, 24h)
text: string
tags: list of strings
date: string (YYYY-MM-DD)

----------------------------------------
4. Write Rules
----------------------------------------
File Creation:
If YYYY-MM.md doesn’t exist, create it with:
# {Month Name} {Year}

Date Section Creation:
If the section for today doesn’t exist, append:
## YYYY-MM-DD

Adding Entries:
Append new entries under the current date section.

Default formats:
- [x] [HH:MM] <text> <#tags...>   (done)
- [ ] [HH:MM] <text> <#tags...>   (todo)

Toggling Status:
Replace [ ] ↔ [x]. Preserve text, tags, and timestamp.

Editing Entries:
- The app may modify text, tags, or time of an existing line.
- Must preserve the same structure and section.
- Edits must be atomic (write to temp, then replace).

Example:
Before: - [x] [09:45] Fixed layout #ui #bug
After:  - [x] [09:50] Fixed loan summary layout #ui #bug #frontend

Deleting Entries:
- The app may remove specific lines entirely.
- Empty sections are allowed (no entries for a date).

Ordering:
- Maintain chronological order of entries within each date.
- The app should not reorder unless user explicitly sorts.

----------------------------------------
5. Read Rules
----------------------------------------
Parsing:
- Maintain current date context (## YYYY-MM-DD)
- Parse lines matching schema regex
- Skip invalid lines silently
- Preserve original order

Parsed Model:
{
  "date": "2025-11-02",
  "time": "09:45",
  "text": "Fixed loan summary layout",
  "tags": ["ui", "bug"],
  "status": "done"
}

----------------------------------------
6. Viewing and Navigation
----------------------------------------
Commands:
kerja today        → show or add to today’s section
kerja prev         → view previous date’s section
kerja next         → view next date’s section
kerja list --week  → aggregate past 7 days
kerja search <term>→ search by keyword or tag
kerja jump <date>  → jump to specific date
kerja toggle <index> [--date]
  → flip todo/done status for the indexed entry (defaults to today)
kerja edit <index> [text ... #tags] [--status] [--time] [--tag] [--date]
  → update the indexed entry; text and #tags may be provided positionally or via flags
kerja delete <index> [--date]
  → remove the indexed entry from the section

Command details:
- Flags accept ISO 8601 dates (`YYYY-MM-DD`) and 24h time (`HH:MM`) in the local timezone.
- `list` defaults to a single day; `--week` maps to `--days=7`. Missing dates in the window are skipped.
- `search` scans the month anchored by the provided `--date` (default: today). Supplying `#tag` searches tags by exact match.
- Commands render friendly output like `[todo] 09:15 Draft doc (#docs)`. Empty sections print `(no entries)`.

----------------------------------------
7. File Ownership
----------------------------------------
Allowed in App / Externally:
Append entry: ✅ / ✅
Toggle done/todo: ✅ / ✅
Edit text: ✅ / ✅
Delete entry: ✅ / ✅
Reorder entries: ❌ / ✅
Rename/move files: ❌ / ✅

----------------------------------------
8. Safety and Performance
----------------------------------------
- Max file size: ~5 MB per month
- All writes atomic (temp write + rename)
- UTF-8 encoding
- Parser must tolerate incomplete files

----------------------------------------
9. Example Workflow
----------------------------------------
kerja log "Fixed loan detail layout" #ui #bug
→ writes: - [x] [09:42] Fixed loan detail layout #ui #bug

kerja todo "Review onboarding PR" #review
→ writes: - [ ] [09:43] Review onboarding PR #review

kerja edit 2 "Updated text" #newtag
→ replaces text of 2nd entry

kerja delete 3
→ removes 3rd entry from section

kerja prev
→ opens previous date section

----------------------------------------
10. Future Extensibility
----------------------------------------
Supports optional YAML front matter, nested headings (### Notes), and export formats (JSON, CSV, HTML).

----------------------------------------
11. Implementation Notes
----------------------------------------
- Writing operations append, edit, or toggle; never reorder.
- Parser supports incremental reads.
- Recommended classes:
  FileManager → file creation & path logic
  Parser → reads and tokenizes Markdown
  Writer → appends, edits, and deletes entries atomically
