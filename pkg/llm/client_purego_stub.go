//go:build !gguf

package llm

import "fmt"

func newPuregoClient(model string) (LLM, error) {
	return nil, fmt.Errorf("purego client not available: build with -tags gguf")
}
