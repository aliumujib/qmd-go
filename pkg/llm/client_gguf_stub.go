//go:build !gguf || !cgo_gguf

package llm

import "fmt"

func newGGUFClient(model string) (LLM, error) {
	return nil, fmt.Errorf("GGUF CGO fallback not built: build with -tags gguf,cgo_gguf or provide libllama_go for purego")
}
