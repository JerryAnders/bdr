# bdr — Semantic Memory for Beads

## Overview

`bdr` is a standalone CLI sidecar tool that adds **semantic memory** to Beads (`bd`).
It solves the missing read path: `bd remember` writes memories, `bdr recall` retrieves
the most relevant ones using vector similarity search — finding memories by *meaning*,
not keywords.

`bdr` is **not a fork of Beads**. It is an independent tool that reads from Beads
via the `bd memories --json` command and maintains its own vector index alongside
the Beads database in the `.beads/` directory.

---

## Problem Statement

Beads stores agent memories via `bd remember "<text>"`. The only way to retrieve
them today is `bd memories` or `bd prime`, which dumps all memories into context.
This does not scale — a project with 200+ memories cannot afford to load all of
them on every agent action, and keyword search misses semantically related memories
that use different words.

`bdr recall "<query>"` retrieves only the top-N semantically relevant memories
using natural language meaning rather than exact keyword matching.

---

## Design Principles

- **Zero external server dependencies** — no Ollama, no separate process, no daemon
- **Zero CGO** — pure Go, no C/C++ compiler required, no shared library to install
- **Zero Dolt dependency** — reads from `bd memories --json`, never connects to Dolt directly
- **Lean install** — one binary + ONNX model downloaded on `bdr init`
- **Independent of Beads internals** — uses only the public `bd` CLI interface
- **Self-healing index** — `bdr recall` auto-syncs before searching, no daemon needed
- **Gitignored artifacts** — vector index is derived and never committed to git

---

## Data Source — Verified

`bd memories --json` returns clean JSON key-value pairs:

```json
{
  "tag-tiptap-toolbar-buttons-must-use-onmousedown": "Tiptap toolbar buttons must use onMouseDown...",
  "never-commit-or-push-code-changes-until-the": "Never commit or push code changes...",
  "tag-architecture-artmojo-next-js-vercel-frontend-clerk": "tag:architecture Artmojo: Next.js..."
}
```

This is the sole data source for `bdr`. No Dolt connection, no SQL queries, no port
management. Verified working in embedded Beads mode.

---

## Architecture

```
bd remember "some decision"           ← Beads writes memory to Dolt (unchanged)
        ↓
bdr recall "<query>"                  ← agent calls this
        ↓
exec: bd memories --json              ← shell out to get all current memories
        ↓
compare keys to index state           ← find new/missing keys since last sync
        ↓
embed new memories via hugot          ← pure Go, no CGO, no Ollama
        ↓
upsert into chromem-go index          ← persisted to .beads/bdr-index/
        ↓
embed query → cosine similarity       ← search
        ↓
return top-N memory values            ← printed to stdout for agent
```

---

## Commands

### `bdr init`

One-time setup per machine. Run once after `bd init` in a project.

**Actions:**
1. Detect `.beads/` directory in current working directory — exit with clear error if not found
2. Download ONNX embedding model to `~/.bdr/models/` (global, shared across projects)
   - Model: `sentence-transformers/all-MiniLM-L6-v2` in ONNX format (~22MB)
   - Source: HuggingFace Hub via `go-huggingface` downloader
   - Skip download if model already exists at `~/.bdr/models/`
3. Build initial vector index from `bd memories --json`
   - Embed all memories using `hugot` pure Go backend
   - Persist index to `.beads/bdr-index/` via `chromem.NewPersistentDB`
4. Add `.beads/bdr-index/` to `.beads/.gitignore` (create `.gitignore` if absent)
5. Print summary: `Indexed N memories. Ready.`

**Warning if no Dolt remote configured:**
```
Warning: no Dolt remote configured (run 'bd dolt remote list' to check).
Index reflects local memories only. Cross-machine sync requires 'bd dolt push/pull'.
```

---

### `bdr recall "<query>"`

Retrieve semantically relevant memories for a natural language query.

**Actions:**
1. Shell out: `bd memories --json` — exit with clear error if `bd` not found or fails
2. Compare returned keys against keys recorded in chromem-go collection
3. Embed and index any new keys (incremental sync — Option B pattern)
4. Embed the query using `hugot` pure Go backend
5. Query chromem-go for top-N results by cosine similarity
6. Print results to stdout

**Default output:**
```
[tag-tiptap-toolbar-buttons] Tiptap toolbar buttons must use onMouseDown + e.preventDefault()...
[tag-pipeline-revalidate] Next.js post page has revalidate=3600 cache. Must call revalidatePath...
[tag-architecture-watermark] Watermark applied at upload time (not by watermark-worker)...
```

**JSON output (`--json` flag):**
```json
[
  {
    "key": "tag-tiptap-toolbar-buttons-must-use-onmousedown",
    "value": "Tiptap toolbar buttons must use onMouseDown...",
    "score": 0.91
  }
]
```

**Flags:**
- `--top N` — return top N results (default: 5)
- `--json` — machine-readable JSON output
- `--min-score F` — minimum similarity threshold 0.0–1.0 (default: 0.0)

---

## Technology Stack

| Concern | Choice | Rationale |
|---|---|---|
| Language | Go | Matches Beads ecosystem |
| Embeddings | `knights-analytics/hugot` pure Go backend | Zero CGO, zero Ollama, macOS ARM ✓ |
| Embedding model | `sentence-transformers/all-MiniLM-L6-v2` ONNX | Industry standard, ~22MB, verified with hugot |
| Vector store | `philippgille/chromem-go` | Pure Go, embeddable, persistent, zero deps |
| Index persistence | `chromem.NewPersistentDB(".beads/bdr-index/")` | Per project, gitignored |
| Model storage | `~/.bdr/models/` | Global per machine, shared across projects |
| Model download | `gomlx/go-huggingface` hub downloader | Clean HuggingFace integration in Go |
| Memory source | `bd memories --json` | Public CLI interface, works in all Beads modes |

**Why not `kelindar/search`:**
Requires precompiled llama.cpp shared library binaries. No macOS precompiled binary
provided — macOS users must compile llama.cpp from source (requires CMake, C++ compiler).
Violates the lean install principle.

**Why not Ollama:**
External server dependency. Requires separate install and a running process.
Violates the zero external server dependency principle.

**Why `hugot` pure Go backend:**
No CGO, no shared libraries, no ONNX Runtime install. Specifically designed for
smaller models like `all-MiniLM-L6-v2`. Works on macOS ARM, Linux, Windows.
Slower than native ONNX Runtime but negligible for datasets under 1,000 memories.

**Why `chromem-go`:**
Pure Go, zero third-party dependencies, embeddable (no server), optional persistence
via gob files, cosine similarity search, accepts pre-computed embeddings from hugot.
Query performance: ~0.3ms for 1,000 documents on mid-range CPU.

---

## File Layout

```
~/.bdr/
  models/
    all-MiniLM-L6-v2/           ← ONNX model files, downloaded once globally
      model.onnx
      tokenizer.json
      config.json

<project>/
  .beads/
    bdr-index/                  ← chromem-go persistent index (gitignored)
    .gitignore                  ← bdr init adds bdr-index/ here
```

---

## CLAUDE.md Integration

Add these rules to the project `CLAUDE.md` after running `bdr init`:

```markdown
## Semantic Memory (bdr)
- Before claiming an issue, run `bdr recall "<issue title and description>"`
  to retrieve relevant past decisions and context.
- When a question arises mid-task, run `bdr recall "<your question>"`
  before proceeding.
- After every commit, run `bd remember "<summary of key decision or context>"`
  to preserve important decisions for future sessions.
```

---

## Sync Behavior

| Scenario | Behavior |
|---|---|
| Single developer, single machine | Auto-sync on every `bdr recall`. Always current. |
| Single developer, no Dolt remote | Works correctly. Local memories only. |
| Multiple developers, Dolt remote configured | Each developer runs `bdr init`. Index rebuilt from local `bd memories --json`. Correct if memories synced via `bd dolt push/pull`. |
| Multiple developers, no Dolt remote | Each developer has independent index. Beads sync concern, not a `bdr` concern. |

`bdr` inherits whatever sync guarantees Beads provides. It does not solve
cross-machine memory sync — that is Beads' responsibility.

---

## Error Handling

| Condition | Behavior |
|---|---|
| `.beads/` not found | Exit 1: `error: no beads project found. Run 'bd init' first.` |
| `bd` not in PATH | Exit 1: `error: 'bd' command not found. Install beads first.` |
| `bd memories --json` fails | Exit 1: print bd's error output |
| Model not downloaded | Exit 1: `error: model not found. Run 'bdr init' first.` |
| No memories in Beads | Warn and exit 0: `no memories found. Use 'bd remember' to add some.` |
| Zero results above min-score | Print empty result with note |

---

## Go Module

```
module github.com/JerryAnders/bdr

go 1.22

require (
    github.com/knights-analytics/hugot  latest
    github.com/philippgille/chromem-go  latest
    github.com/gomlx/go-huggingface     latest
)
```

---

## MVP Scope

The MVP delivers exactly two commands: `bdr init` and `bdr recall`. Nothing else.

Out of scope for MVP:
- `bdr sync` — explicit re-index command
- `bdr stats` — index statistics
- `bdr forget "<key>"` — remove memory from index
- `bdr status` — show index health
- Homebrew formula

---

## Installation (MVP)

```bash
# Prerequisites: Go 1.22+, beads (bd) installed
go install github.com/JerryAnders/bdr@latest

# One-time per machine — downloads ONNX model (~22MB):
bdr init

# Usage (run from project root):
bdr recall "how did we handle watermarks?"
bdr recall "what was the tiptap fix?" --top 3
bdr recall "deployment lessons" --json
```

---

## Verified Decisions

| Question | Decision | Verified |
|---|---|---|
| Fork Beads or sidecar? | Sidecar — fully independent | ✓ |
| Embedding library? | hugot pure Go backend | ✓ macOS ARM, no CGO |
| Vector store? | chromem-go | ✓ pure Go, persistent |
| Model format? | ONNX (not GGUF) | ✓ required by hugot |
| Model delivery? | Downloaded on bdr init from HuggingFace | ✓ |
| Data source? | bd memories --json | ✓ verified working |
| Index freshness? | Auto-sync on every bdr recall | ✓ |
| Gitignore index? | Yes — bdr init adds bdr-index/ | ✓ |
| Tool name? | bdr | ✓ |
| No Ollama? | Confirmed — hugot pure Go needs no Ollama | ✓ |
| No CGO? | Confirmed — hugot pure Go backend | ✓ |
| macOS ARM support? | Confirmed — hugot pure Go runs everywhere | ✓ |
