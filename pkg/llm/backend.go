package llm

import (
	"os"
	"strings"
)

// isGGUFSpec returns true if model looks like a GGUF spec: local path ending in .gguf,
// "repo:file.gguf" (Hugging Face), or "org/repo/file.gguf".
func isGGUFSpec(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	if strings.HasSuffix(model, ".gguf") {
		return true
	}
	if idx := strings.Index(model, ":"); idx >= 0 {
		file := model[idx+1:]
		return strings.TrimSpace(file) != "" && strings.HasSuffix(file, ".gguf")
	}
	return false
}

// NewEmbedClient returns an LLM implementation for embeddings.
// When built with -tags gguf, tries purego first, then falls back to go-llama.cpp if purego lib not found.
// Otherwise use GGUF if QMD_EMBED_BACKEND=gguf or model is a GGUF spec. Uses Ollama/OpenAI API otherwise.
func NewEmbedClient(model string) (LLM, error) {
	backend := os.Getenv("QMD_EMBED_BACKEND")
	useGGUF := backend == "gguf" || isGGUFSpec(model) || (GGUFEnabled() && backend != "api")
	if useGGUF {
		// Try purego first (kelindar/search method)
		if client, err := newPuregoClient(model); err == nil {
			return client, nil
		}
		// Fallback to go-llama.cpp (CGO method)
		return newGGUFClient(model)
	}
	baseURL := os.Getenv("OLLAMA_HOST")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	return NewOpenAIClient(baseURL, model), nil
}
