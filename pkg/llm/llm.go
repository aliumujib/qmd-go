package llm

type EmbeddingResult struct {
	Embedding []float32
	Model     string
}

type LLM interface {
	Embed(text string) (*EmbeddingResult, error)
	Generate(prompt string) (string, error)
}
