package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	chromem "github.com/philippgille/chromem-go"
)

const (
	collectionName = "memories"
	keysFile       = "keys.json"
)

// Result holds a single recall result.
type Result struct {
	Key   string
	Value string
	Score float32
}

// DB wraps a chromem-go persistent database plus a side-car keys index.
type DB struct {
	db         *chromem.DB
	collection *chromem.Collection
	indexDir   string
	keys       map[string]struct{} // in-memory keys mirror
}

// Open opens (or creates) a persistent DB at the given directory.
func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create index dir: %w", err)
	}

	db, err := chromem.NewPersistentDB(dir, false)
	if err != nil {
		return nil, fmt.Errorf("open vector store: %w", err)
	}

	col, err := db.GetOrCreateCollection(collectionName, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get collection: %w", err)
	}

	keys, err := loadKeys(dir)
	if err != nil {
		return nil, err
	}

	return &DB{db: db, collection: col, indexDir: dir, keys: keys}, nil
}

// Upsert adds or updates a memory by key with a pre-computed embedding.
func (d *DB) Upsert(key, value string, embedding []float32) error {
	doc := chromem.Document{
		ID:        key,
		Content:   value,
		Embedding: embedding,
	}
	if err := d.collection.AddDocument(context.Background(), doc); err != nil {
		return err
	}
	d.keys[key] = struct{}{}
	return saveKeys(d.indexDir, d.keys)
}

// Keys returns all document keys currently in the collection.
func (d *DB) Keys() ([]string, error) {
	out := make([]string, 0, len(d.keys))
	for k := range d.keys {
		out = append(out, k)
	}
	return out, nil
}

// Query returns the top-N most similar memories to the given embedding.
func (d *DB) Query(embedding []float32, topN int, minScore float32) ([]Result, error) {
	if d.collection.Count() == 0 {
		return nil, nil
	}
	n := topN
	if n > d.collection.Count() {
		n = d.collection.Count()
	}
	results, err := d.collection.QueryEmbedding(context.Background(), embedding, n, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("vector query: %w", err)
	}

	var out []Result
	for _, r := range results {
		if r.Similarity < minScore {
			continue
		}
		out = append(out, Result{Key: r.ID, Value: r.Content, Score: r.Similarity})
	}
	return out, nil
}

func loadKeys(dir string) (map[string]struct{}, error) {
	path := filepath.Join(dir, keysFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read keys file: %w", err)
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse keys file: %w", err)
	}
	m := make(map[string]struct{}, len(list))
	for _, k := range list {
		m[k] = struct{}{}
	}
	return m, nil
}

func saveKeys(dir string, keys map[string]struct{}) error {
	list := make([]string, 0, len(keys))
	for k := range keys {
		list = append(list, k)
	}
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, keysFile), data, 0644)
}
