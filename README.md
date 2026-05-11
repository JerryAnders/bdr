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

## Smoke Test

Run this to verify everything works end-to-end:

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
# but a vector search should surface it because the concepts are closely related
bdr recall "release process"
```

Expected output:

```
[always-deploy-to-staging-before-production] always deploy to staging before production
[the-vector-index-lives-at-beads-bdr-index] the vector index lives at .beads/bdr-index/ and is gitignored
```

If `bdr recall "release process"` surfaces the staging memory without any shared keywords, vector similarity is working correctly — it found the memory by *meaning*, not by matching words.

## Usage

```bash
# One-time setup per project (run after bd init)
bdr init

# Retrieve semantically relevant memories
bdr recall "how did we handle watermarks?"
bdr recall "what was the tiptap fix?" --top 3
bdr recall "deployment lessons" --json
```

### Flags for `bdr recall`

| Flag | Default | Description |
|---|---|---|
| `--top N` | 5 | Number of results to return |
| `--json` | false | Output as JSON array with scores |
| `--min-score F` | 0.0 | Minimum similarity threshold (0.0–1.0) |

## How It Works

`bdr recall` shells out to `bd memories --json`, incrementally syncs any new memories into a local vector index, embeds your query, and returns the top-N results by cosine similarity. No daemon, no external server, no CGO.

- **Model**: `KnightsAnalytics/all-MiniLM-L6-v2` (ONNX, ~22MB), downloaded once to `~/.bdr/models/`
- **Index**: stored at `.beads/bdr-index/` (gitignored), rebuilt automatically as you add memories

See [SPEC.md](SPEC.md) for full design details.
