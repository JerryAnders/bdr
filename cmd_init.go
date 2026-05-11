package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/JerryAnders/bdr/internal/bd"
	"github.com/JerryAnders/bdr/internal/embed"
	"github.com/JerryAnders/bdr/internal/store"
	"github.com/JerryAnders/bdr/internal/workspace"
)

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: bdr init")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	beadsDir, indexDir, err := workspace.Detect(cwd)
	if err != nil {
		return err
	}
	_ = beadsDir

	modelDir, err := embed.EnsureModel()
	if err != nil {
		return err
	}

	memories, err := bd.Memories()
	if err != nil {
		return err
	}
	if len(memories) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no memories found. Use 'bd remember' to add some.")
		return nil
	}

	engine, err := embed.NewEngine(modelDir)
	if err != nil {
		return err
	}
	defer engine.Close()

	db, err := store.Open(indexDir)
	if err != nil {
		return err
	}

	keys := make([]string, 0, len(memories))
	vals := make([]string, 0, len(memories))
	for k, v := range memories {
		keys = append(keys, k)
		vals = append(vals, v)
	}

	embeddings, err := engine.Embed(vals)
	if err != nil {
		return err
	}

	for i, key := range keys {
		if err := db.Upsert(key, vals[i], embeddings[i]); err != nil {
			return err
		}
	}

	if err := workspace.EnsureGitignore(beadsDir); err != nil {
		return err
	}

	warnIfNoDoltRemote()

	fmt.Printf("Indexed %d memories. Ready.\n\n", len(memories))
	printClaudeMdInstructions()
	return nil
}

const claudeMdSnippet = `## Semantic Memory (bdr)
- Before claiming an issue, run ` + "`" + `bdr recall "<issue title and description>"` + "`" + `
  to retrieve relevant past decisions and context.
- When a question arises mid-task, run ` + "`" + `bdr recall "<your question>"` + "`" + `
  before proceeding.
- After every commit, run ` + "`" + `bd remember "<summary of key decision or context>"` + "`" + `
  to preserve important decisions for future sessions.`

func printClaudeMdInstructions() {
	fmt.Println("To enable semantic memory in agents, add the following to your CLAUDE.md and/or AGENTS.md:")
	fmt.Println()
	fmt.Println(claudeMdSnippet)
}

func warnIfNoDoltRemote() {
	// Check if a Dolt remote is configured; warn if not.
	// Non-fatal: missing remote is a valid single-machine configuration.
	out, err := runCommand("bd", "dolt", "remote", "list")
	if err != nil || len(out) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no Dolt remote configured (run 'bd dolt remote list' to check).")
		fmt.Fprintln(os.Stderr, "  Index reflects local memories only. Cross-machine sync requires 'bd dolt push/pull'.")
	}
}
