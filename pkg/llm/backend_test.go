package llm

import "testing"

func TestDefaultEmbedModel(t *testing.T) {
	model := DefaultEmbedModel()
	if model == "" {
		t.Fatal("DefaultEmbedModel() must not be empty")
	}
	if GGUFEnabled() {
		// Default GGUF model is EmbeddingGemma 300M (Tobi's qmd setup)
		if model != "ggml-org/embeddinggemma-300M-GGUF:embeddinggemma-300M-Q8_0.gguf" {
			t.Errorf("GGUF build: expected EmbeddingGemma 300M HF spec, got %q", model)
		}
	} else {
		if model != "nomic-embed-text" {
			t.Errorf("API build: expected nomic-embed-text, got %q", model)
		}
	}
}

func TestGGUFEnabled(t *testing.T) {
	// Just ensure it returns a bool; actual value depends on build tag
	_ = GGUFEnabled()
}
