//go:build !gguf

package llm

import "fmt"

func newGGUFClient(model string) (LLM, error) {
	return nil, fmt.Errorf("GGUF backend not built: build with -tags gguf (requires go-llama.cpp and libbinding.a)")
}
