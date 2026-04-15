package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type OpenAIClient struct {
	BaseURL string
	APIKey  string
	Model   string
}

func NewOpenAIClient(baseURL, model string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1" // Default to Ollama
	}
	return &OpenAIClient{
		BaseURL: baseURL,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   model,
	}
}

type embeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (c *OpenAIClient) Embed(text string) (*EmbeddingResult, error) {
	reqBody := embeddingRequest{
		Input: text,
		Model: c.Model,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/embeddings", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var res embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	if len(res.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return &EmbeddingResult{
		Embedding: res.Data[0].Embedding,
		Model:     c.Model,
	}, nil
}

func (c *OpenAIClient) Generate(prompt string) (string, error) {
	// Stub implementation for Generate
	return "", fmt.Errorf("not implemented")
}
