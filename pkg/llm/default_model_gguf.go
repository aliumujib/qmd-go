//go:build gguf

package llm

// DefaultEmbedModel returns the default embedding model for GGUF builds.
// Matches Tobi's qmd: EmbeddingGemma 300M (ggml-org). Requires llama.cpp with
// gemma-embedding support (e.g. make update-llama-cpp-ggml). If load fails use
// QMD_EMBED_BACKEND=api or set QMD_EMBED_MODEL to a Nomic spec.
func DefaultEmbedModel() string {
	return "ggml-org/embeddinggemma-300M-GGUF:embeddinggemma-300M-Q8_0.gguf"
}
