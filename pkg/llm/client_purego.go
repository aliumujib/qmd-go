//go:build gguf

package llm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"github.com/aliumujib/qmd-go/pkg/huggingface"
	"github.com/ebitengine/purego"
)

// puregoClient wraps llama_go shared library via purego (no CGO).
type puregoClient struct {
	mu         sync.Mutex
	model      string
	modelPtr   unsafe.Pointer
	lib        uintptr
	nDims      int
	loadFn     func(path *byte, nCtx, nGpuLayers int) unsafe.Pointer
	freeFn     func(model unsafe.Pointer)
	embedFn    func(model unsafe.Pointer, text *byte, embedding *float32, maxDims int) int
	getErrorFn func() string
}

var (
	libPath     string
	libPathOnce sync.Once
)

func findLibPath() string {
	libPathOnce.Do(func() {
		// Try build directory first (for development)
		candidates := []string{
			"llama-go/build/libllama_go.so",    // Linux build dir
			"llama-go/build/llama_go.dll",      // Windows build dir
			"llama-go/build/libllama_go.dylib", // macOS build dir
			"libllama_go.so",                   // Linux current dir
			"llama_go.dll",                     // Windows current dir
			"libllama_go.dylib",                // macOS current dir
			"/usr/lib/libllama_go.so",          // Linux system
			"/usr/local/lib/libllama_go.so",
		}
		if env := os.Getenv("LLAMA_GO_LIB"); env != "" {
			candidates = append([]string{env}, candidates...)
		}
		for _, cand := range candidates {
			if _, err := os.Stat(cand); err == nil {
				libPath = cand
				return
			}
		}
		// Try relative to executable
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			for _, name := range []string{"libllama_go.so", "llama_go.dll", "libllama_go.dylib"} {
				path := filepath.Join(exeDir, name)
				if _, err := os.Stat(path); err == nil {
					libPath = path
					return
				}
			}
		}
	})
	return libPath
}

func newPuregoClient(model string) (LLM, error) {
	libPath := findLibPath()
	if libPath == "" {
		return nil, fmt.Errorf("llama_go shared library not found (set LLAMA_GO_LIB or place libllama_go.so in PATH)")
	}

	lib, err := purego.Dlopen(libPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("dlopen %s: %w", libPath, err)
	}

	var loadFn func(path *byte, nCtx, nGpuLayers int) unsafe.Pointer
	var freeFn func(model unsafe.Pointer)
	var embedFn func(model unsafe.Pointer, text *byte, embedding *float32, maxDims int) int
	var getErrorFn func() string

	purego.RegisterLibFunc(&loadFn, lib, "llama_go_load")
	purego.RegisterLibFunc(&freeFn, lib, "llama_go_free")
	purego.RegisterLibFunc(&embedFn, lib, "llama_go_embed")
	purego.RegisterLibFunc(&getErrorFn, lib, "llama_go_get_error")

	ctx := context.Background()
	path, err := huggingface.ResolveModel(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("resolve GGUF model: %w", err)
	}

	if fi, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("model file not found: %s: %w", path, err)
	} else if fi.Size() == 0 {
		return nil, fmt.Errorf("model file is empty: %s", path)
	}

	pathBytes := []byte(path)
	pathBytes = append(pathBytes, 0)           // null terminator
	modelPtr := loadFn(&pathBytes[0], 2048, 0) // n_ctx=2048, n_gpu_layers=0
	if modelPtr == nil {
		if errMsg := getErrorFn(); errMsg != "" {
			return nil, fmt.Errorf("load model: %s", errMsg)
		}
		return nil, fmt.Errorf("load model failed")
	}

	// Probe embedding dimensions by embedding empty string (or a test string)
	testText := "test"
	testBytes := []byte(testText)
	testBytes = append(testBytes, 0)
	var probe [1024]float32 // max reasonable dims
	nDims := embedFn(modelPtr, &testBytes[0], &probe[0], len(probe))
	if nDims <= 0 {
		freeFn(modelPtr)
		if errMsg := getErrorFn(); errMsg != "" {
			return nil, fmt.Errorf("probe embedding dims: %s", errMsg)
		}
		return nil, fmt.Errorf("probe embedding dims failed")
	}

	return &puregoClient{
		model:      model,
		modelPtr:   modelPtr,
		lib:        lib,
		nDims:      nDims,
		loadFn:     loadFn,
		freeFn:     freeFn,
		embedFn:    embedFn,
		getErrorFn: getErrorFn,
	}, nil
}

func (c *puregoClient) Embed(text string) (*EmbeddingResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	textBytes := []byte(text)
	textBytes = append(textBytes, 0) // null terminator

	embedding := make([]float32, c.nDims)
	nDims := c.embedFn(c.modelPtr, &textBytes[0], &embedding[0], len(embedding))
	if nDims <= 0 {
		if errMsg := c.getErrorFn(); errMsg != "" {
			return nil, fmt.Errorf("embed: %s", errMsg)
		}
		return nil, fmt.Errorf("embedding failed")
	}

	if nDims != c.nDims {
		embedding = embedding[:nDims]
	}

	return &EmbeddingResult{
		Embedding: embedding,
		Model:     c.model,
	}, nil
}

func (c *puregoClient) Generate(prompt string) (string, error) {
	return "", fmt.Errorf("Generate not implemented for purego embedding client")
}

func (c *puregoClient) Close() error {
	if c.modelPtr != nil {
		c.freeFn(c.modelPtr)
		c.modelPtr = nil
	}
	return nil
}
