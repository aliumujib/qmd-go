//go:build !gguf

package llm

// DefaultEmbedModel returns the default embedding model for API builds (Ollama/OpenAI).
func DefaultEmbedModel() string {
	return "nomic-embed-text"
}
