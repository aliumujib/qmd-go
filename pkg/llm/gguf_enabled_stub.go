//go:build !gguf

package llm

// GGUFEnabled reports whether this binary was built with GGUF support.
func GGUFEnabled() bool {
	return false
}
