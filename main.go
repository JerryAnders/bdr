package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	case "recall":
		if err := runRecall(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`bdr — semantic memory for beads

Usage:
  bdr init                  one-time setup: download model, build index
  bdr recall "<query>"      retrieve semantically relevant memories

Options for recall:
  --top N        return top N results (default: 5)
  --json         output as JSON array
  --min-score F  minimum similarity threshold 0.0–1.0 (default: 0.0)
`)
}
