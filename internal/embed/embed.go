package embed

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

const (
	// HuggingFace repo with pre-built ONNX files (single model.onnx, no CGO).
	modelRepo = "KnightsAnalytics/all-MiniLM-L6-v2"
	// Derived local directory name after hugot.DownloadModel runs.
	modelDirName = "KnightsAnalytics_all-MiniLM-L6-v2"
)

// ModelDir returns the expected path to the downloaded model directory.
// Does not check whether it exists.
func ModelDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".bdr", "models", modelDirName), nil
}

// EnsureModel downloads the ONNX model to ~/.bdr/models/ if not already present.
// Returns the path to the model directory.
func EnsureModel() (string, error) {
	modelDir, err := ModelDir()
	if err != nil {
		return "", err
	}

	onnxPath := filepath.Join(modelDir, "model.onnx")
	if _, err := os.Stat(onnxPath); err == nil {
		return modelDir, nil // already downloaded
	}

	destDir := filepath.Dir(modelDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create model dir: %w", err)
	}

	fmt.Printf("Downloading embedding model %s (~22MB)...\n", modelRepo)
	opts := hugot.NewDownloadOptions()
	opts.Verbose = true
	downloaded, err := hugot.DownloadModel(context.Background(), modelRepo, destDir, opts)
	if err != nil {
		return "", fmt.Errorf("download model: %w", err)
	}
	return downloaded, nil
}

// Engine wraps a hugot feature extraction pipeline for sentence embedding.
type Engine struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
}

// NewEngine loads the ONNX model at modelDir and returns a ready Engine.
func NewEngine(modelDir string) (*Engine, error) {
	session, err := hugot.NewGoSession(context.Background())
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	config := hugot.FeatureExtractionConfig{
		ModelPath:    modelDir,
		Name:         "bdr-embed",
		OnnxFilename: "model.onnx",
		Options: []hugot.FeatureExtractionOption{
			pipelines.WithNormalization(),
		},
	}
	pipe, err := hugot.NewPipeline(session, config)
	if err != nil {
		_ = session.Destroy()
		return nil, fmt.Errorf("load embedding pipeline: %w", err)
	}

	return &Engine{session: session, pipeline: pipe}, nil
}

// Embed returns one embedding vector per input string.
func (e *Engine) Embed(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out, err := e.pipeline.RunPipeline(context.Background(), texts)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	return out.Embeddings, nil
}

// Close releases all resources held by the engine.
func (e *Engine) Close() {
	_ = e.session.Destroy()
}
