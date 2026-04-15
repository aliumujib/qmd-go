//go:build gguf

package llm

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/aliumujib/qmd-go/pkg/huggingface"
	llama "github.com/go-skynet/go-llama.cpp"
)

// ggufClient wraps go-llama.cpp for embedding-only use. Thread-safe.
type ggufClient struct {
	mu    sync.Mutex
	model string
	llama *llama.LLama
}

func newGGUFClient(model string) (LLM, error) {
	ctx := context.Background()
	path, err := huggingface.ResolveModel(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("resolve GGUF model: %w", err)
	}
	// Verify file exists and is readable
	if fi, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("model file not found: %s: %w", path, err)
	} else if fi.Size() == 0 {
		return nil, fmt.Errorf("model file is empty: %s", path)
	}
	l, err := llama.New(path, llama.EnableEmbeddings, llama.SetContext(2048))
	if err != nil {
		return nil, fmt.Errorf("load GGUF model from %s: %w (hint: use QMD_EMBED_BACKEND=api to use Ollama for embeddings)", path, err)
	}
	return &ggufClient{model: model, llama: l}, nil
}

func (c *ggufClient) Embed(text string) (*EmbeddingResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	vec, err := c.llama.Embeddings(text)
	if err != nil {
		return nil, err
	}
	// Trim to actual size (llama may return oversized slice)
	if vec == nil {
		return nil, fmt.Errorf("no embedding returned")
	}
	return &EmbeddingResult{
		Embedding: vec,
		Model:     c.model,
	}, nil
}

func (c *ggufClient) Generate(prompt string) (string, error) {
	return "", fmt.Errorf("Generate not implemented for GGUF embedding client")
}
