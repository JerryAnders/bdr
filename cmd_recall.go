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
	minScore := fs.Float64("min-score", 0.0, "minimum similarity threshold (0.0–1.0)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: bdr recall "<query>" [--top N] [--json] [--min-score F]`)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("query argument required")
	}
	query := fs.Arg(0)

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
	printText(results)
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

func printText(results []store.Result) {
	const maxLen = 120
	for _, r := range results {
		val := r.Value
		if len(val) > maxLen {
			val = val[:maxLen] + "..."
		}
		fmt.Printf("[%s] %s\n", r.Key, val)
	}
}
