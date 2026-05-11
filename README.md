# bdr — Semantic Memory for Beads

`bdr` adds semantic memory recall to [Beads](https://beads.sh). `bd remember` writes memories; `bdr recall` retrieves the most relevant ones using vector similarity — finding memories by *meaning*, not keywords.

## Prerequisites

You need both `bd` and `bdr` on your PATH:

```bash
which bd    # should resolve, e.g. /opt/homebrew/bin/bd
which bdr   # should resolve after go install (see Installation)
```

If `bd` is missing, install Beads first. If `bdr` is missing, see Installation below.

## Installation

```bash
# Prerequisites: Go 1.22+, beads (bd) installed
git clone https://github.com/JerryAnders/bdr
cd bdr
go install .
```

## Smoke Test: Tool

Verify the tool itself works end-to-end:

```bash
mkdir /tmp/test-bdr && cd /tmp/test-bdr
echo "# test" > README.md
git init
bd init

# Add a few memories
bd remember "always deploy to staging before production"
bd remember "the database password is stored in 1Password under infrastructure"
bd remember "the vector index lives at .beads/bdr-index/ and is gitignored"

# Build the index (downloads ~22MB ONNX model on first run)
bdr init

# Semantic search — "release process" shares no keywords with the staging memory,
# but a vector search should surface it because the concepts are closely related.
bdr recall "release process"
```

Expected output:

```
always deploy to staging before production
```

If `bdr recall "release process"` surfaces the staging memory without any shared keywords, vector similarity is working correctly — it found the memory by *meaning*, not by matching words.

## Smoke Test: Agent Integration

Verify that agents automatically call `bdr recall` before starting work.

**Setup:**

```bash
mkdir /tmp/test-bdr-agent && cd /tmp/test-bdr-agent
git init
bd init

bd remember "always write tests before marking a task complete"
bd remember "use snake_case for all Python function names"
bd remember "deploy to staging.example.com before production"
bd remember "never use print statements for debugging — use the logger"
bd remember "database migrations must be reviewed by a second engineer"

bdr init
```

**Configure CLAUDE.md:**

`bdr init` prints the snippet to add. Your CLAUDE.md should look like this — note the trimmed beads section (no aggressive session-completion mandates) and the explicit `bdr recall` requirement:

```markdown
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking.

` + "```" + `bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
` + "```" + `

- Use `bd` for task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Semantic Memory (bdr)

Before claiming any issue, you MUST run `bdr recall` with the issue title to surface
relevant past decisions and conventions. Do not skip this — it prevents repeating past mistakes.

` + "```" + `bash
bdr recall "<issue title or description>"
` + "```" + `

When a question arises mid-task, run `bdr recall "<your question>"` before proceeding.

After every commit, run `bd remember "<key decision or context>"` to preserve it for future sessions.
```

> **Note on `bd prime`:** If your CLAUDE.md or AGENTS.md references `bd prime`, remove it. `bd prime` dumps all memories into context at session start — the opposite of what `bdr` does. Let `bdr recall` surface memories on demand instead.

**Run the test:**

Open a Claude Code session in the project directory and give it a task that requires creating a beads issue:

```
Create a Python utility module with arithmetic functions and tests
```

**Expected behavior:**

```
bd create ...           ← creates the issue
bdr recall "..."        ← fires automatically before claiming
bd update ... --claim   ← claims the issue
... writes code ...
bd close ...            ← closes the issue
```

The `bdr recall` result should surface the snake_case and testing memories — constraints the agent applies before writing a single line of code.

## Usage

```bash
# One-time setup per project (run after bd init)
bdr init

# Retrieve semantically relevant memories
bdr recall "how did we handle authentication?"
bdr recall "what are the deployment steps?" --top 3
bdr recall "database conventions" --json
bdr recall "naming conventions" --keys   # show memory keys alongside values
```

### Flags for `bdr recall`

| Flag | Default | Description |
|---|---|---|
| `--top N` | 5 | Number of results to return |
| `--min-score F` | 0.2 | Minimum similarity threshold (0.0–1.0) |
| `--json` | false | Output as JSON array with scores |
| `--keys` | false | Prefix each result with its memory key |

## What Goes Where

Not everything belongs in `bd remember`. The right separation:

| What | Where | Why |
|---|---|---|
| Hard rules ("never deploy on Fridays") | CLAUDE.md | Always in context, never missed |
| Non-negotiable constraints | CLAUDE.md | Too important to risk a low similarity score |
| Past decisions and discoveries | `bd remember` / `bdr recall` | Retrieved on demand when relevant |
| Project conventions learned over time | `bd remember` / `bdr recall` | Surfaces when the agent needs them |

This matters because the main risk of `bdr recall` — that a critical memory might not surface for a given query — disappears if critical constraints live in CLAUDE.md instead. `bdr` is only lossy if you ask it to carry things it shouldn't.

## Tradeoffs

`bdr recall` is not a strict upgrade over `bd prime`. Know the limitations before adopting it:

- **Recall is lossy** — it returns the most *similar* memories, not the most *important* ones. "never deploy on Fridays" may score low for an unrelated query and never surface. Mitigate this by keeping hard rules in CLAUDE.md, not in memories (see above).
- **Similarity ≠ relevance** — the embedding model matches on language patterns, not project knowledge. A memory like "the widget service owns user preferences" won't surface for a query like "where should I store this setting?" because the vocabulary doesn't overlap. Write memories in plain, descriptive language that anticipates how you'll search for them later.
- **The index can drift** — if `bd dolt pull` hasn't been run, `bdr recall` reflects local memories only. `bd prime` always reads from the live database.
- **More moving parts** — `bdr` requires a downloaded ONNX model, a persistent index, and a separate binary. `bd prime` has no dependencies beyond beads itself.
- **Context size may not matter** — on small projects with few memories, loading everything via `bd prime` is cheap and complete. `bdr` adds complexity for a problem that may not exist yet.

If any critical memory being silently missed is unacceptable, stick with `bd prime`. `bdr` is the right trade when memory count is large enough that context bloat and staleness outweigh the risk of an occasional miss.

## Why bdr vs `bd prime`

Beads provides `bd prime` to load memories into agent context at session start. It works well for small projects but has two limitations that grow with the project:

- **Staleness** — `bd prime` is a one-shot snapshot. Memories added mid-session via `bd remember` aren't visible until the next session.
- **Context bloat** — it dumps all memories regardless of relevance. On a project with many memories, most of what gets loaded has nothing to do with the current task.

`bdr recall` addresses both: it syncs new memories on every call so it's always current, and it returns only the memories most relevant to the query. Context stays small and focused.

`bd prime` and `bdr` serve the same goal — giving agents access to project knowledge. `bdr` is better suited to projects where the memory count is growing and precision matters more than completeness.

## How It Works

`bdr recall` shells out to `bd memories --json`, incrementally syncs any new memories into a local vector index, embeds your query, and returns the top-N results by cosine similarity. No daemon, no external server, no CGO.

- **Model**: `KnightsAnalytics/all-MiniLM-L6-v2` (ONNX, ~22MB), downloaded once to `~/.bdr/models/`
- **Index**: stored at `.beads/bdr-index/` (gitignored), rebuilt automatically as you add memories

See [SPEC.md](SPEC.md) for full design details.
