# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

`bdr` is a standalone Go CLI sidecar for Beads (`bd`) that adds semantic memory recall via vector similarity search. It reads memories from `bd memories --json` and maintains its own ONNX-backed vector index in `.beads/bdr-index/`. See `SPEC.md` for the full design.

## Commands

```bash
go build ./...          # build
go test ./...           # run all tests
go test ./... -run TestName   # run single test
go vet ./...            # lint
go install .            # install bdr binary
```

## Architecture

Two commands only (MVP scope):

- **`bdr init`** — detects `.beads/` in CWD, downloads ONNX model to `~/.bdr/models/` (once per machine), builds initial vector index from `bd memories --json`, persists to `.beads/bdr-index/`, adds `bdr-index/` to `.beads/.gitignore`
- **`bdr recall "<query>"`** — shells out to `bd memories --json`, incremental-syncs new keys into chromem-go, embeds the query, returns top-N by cosine similarity

Data flow: `bd memories --json` → diff against index keys → embed new memories via hugot → upsert into chromem-go → embed query → cosine search → stdout

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/knights-analytics/hugot` | Pure Go ONNX inference — **no CGO, no Ollama** |
| `github.com/philippgille/chromem-go` | Pure Go embeddable vector store with persistence |
| `github.com/gomlx/go-huggingface` | HuggingFace Hub model downloader |

Model: `sentence-transformers/all-MiniLM-L6-v2` in ONNX format (~22MB), stored at `~/.bdr/models/`.

## Hard Constraints

- **Zero CGO** — hugot pure Go backend only; never use bindings that require a C compiler
- **Zero external server** — no Ollama, no daemon, no separate process
- **Zero Dolt dependency** — only interact with Beads via `bd memories --json` (public CLI)
- Index at `.beads/bdr-index/` is derived/gitignored, model at `~/.bdr/models/` is global/shared

## Go Module

```
module github.com/JerryAnders/bdr
go 1.22
```


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
