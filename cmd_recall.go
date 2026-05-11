package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/JerryAnders/bdr/internal/bd"
	"github.com/JerryAnders/bdr/internal/embed"
	"github.com/JerryAnders/bdr/internal/store"
	"github.com/JerryAnders/bdr/internal/workspace"
)

func runRecall(args []string) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	topN := fs.Int("top", 5, "number of results to return")
	asJSON := fs.Bool("json", false, "output as JSON array")
	showKeys := fs.Bool("keys", false, "prefix each result with its memory key")
	minScore := fs.Float64("min-score", 0.2, "minimum similarity threshold (0.0–1.0)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: bdr recall "<query>" [--top N] [--json] [--keys] [--min-score F]`)
	}

	// Separate the query (first non-flag arg) from flags so that flags may
	// appear before or after the query string.
	var flagArgs, posArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 1 && a[0] == '-' {
			flagArgs = append(flagArgs, a)
			// If this flag takes a value (--top N, --min-score F), consume next arg too.
			name := a
			for name[0] == '-' {
				name = name[1:]
			}
			if eqIdx := len(name); eqIdx > 0 {
				// already has =value inline — no next arg to consume
				if idx := indexOf(name, '='); idx >= 0 {
					continue
				}
			}
			f := fs.Lookup(name)
			if f != nil {
				if _, isBool := f.Value.(boolFlag); !isBool && i+1 < len(args) {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else {
			posArgs = append(posArgs, a)
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if len(posArgs) == 0 {
		fs.Usage()
		return fmt.Errorf("query argument required")
	}
	query := posArgs[0]

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	_, indexDir, err := workspace.Detect(cwd)
	if err != nil {
		return err
	}

	modelDir, err := embed.ModelDir()
	if err != nil {
		return fmt.Errorf("model not found. Run 'bdr init' first.")
	}

	engine, err := embed.NewEngine(modelDir)
	if err != nil {
		return err
	}
	defer engine.Close()

	memories, err := bd.Memories()
	if err != nil {
		return err
	}

	db, err := store.Open(indexDir)
	if err != nil {
		return fmt.Errorf("index not initialized. Run 'bdr init' first.")
	}

	if err := syncNewMemories(db, engine, memories); err != nil {
		return err
	}

	queryEmb, err := engine.Embed([]string{query})
	if err != nil {
		return err
	}

	results, err := db.Query(queryEmb[0], *topN, float32(*minScore))
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "no results found above the minimum score threshold")
		return nil
	}

	if *asJSON {
		return printJSON(results)
	}
	printText(results, *showKeys)
	return nil
}

func syncNewMemories(db *store.DB, engine *embed.Engine, memories map[string]string) error {
	existing, err := db.Keys()
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(existing))
	for _, k := range existing {
		seen[k] = true
	}

	var newKeys, newVals []string
	for k, v := range memories {
		if !seen[k] {
			newKeys = append(newKeys, k)
			newVals = append(newVals, v)
		}
	}
	if len(newKeys) == 0 {
		return nil
	}

	embeddings, err := engine.Embed(newVals)
	if err != nil {
		return err
	}
	for i, key := range newKeys {
		if err := db.Upsert(key, newVals[i], embeddings[i]); err != nil {
			return err
		}
	}
	return nil
}

type jsonResult struct {
	Key   string  `json:"key"`
	Value string  `json:"value"`
	Score float32 `json:"score"`
}

func printJSON(results []store.Result) error {
	out := make([]jsonResult, len(results))
	for i, r := range results {
		out[i] = jsonResult{Key: r.Key, Value: r.Value, Score: r.Score}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

type boolFlag interface {
	IsBoolFlag() bool
}

func indexOf(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func printText(results []store.Result, showKeys bool) {
	const maxLen = 120
	for _, r := range results {
		val := r.Value
		if len(val) > maxLen {
			val = val[:maxLen] + "..."
		}
		if showKeys {
			fmt.Printf("[%s] %s\n", r.Key, val)
		} else {
			fmt.Println(val)
		}
	}
}
