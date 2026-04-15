package huggingface

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DefaultRevision is the default branch to resolve files from.
const DefaultRevision = "main"

// ResolveURL returns the direct download URL for a file in a Hugging Face repo.
// repo is e.g. "ggml-org/embeddinggemma-300M-GGUF", revision e.g. "main", file e.g. "embeddinggemma-300M-Q8_0.gguf".
func ResolveURL(repo, revision, file string) string {
	if revision == "" {
		revision = DefaultRevision
	}
	return fmt.Sprintf("https://huggingface.co/%s/resolve/%s/%s", strings.TrimPrefix(repo, "https://huggingface.co/"), revision, file)
}

// ModelCacheDir returns the directory where GGUF models are cached (~/.cache/qmd/models or QMD_MODEL_CACHE).
func ModelCacheDir() (string, error) {
	if d := os.Getenv("QMD_MODEL_CACHE"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		cache = filepath.Join(home, ".cache")
	}
	return filepath.Join(cache, "qmd", "models"), nil
}

// LocalPath returns the path where a repo/file would be cached.
func LocalPath(repo, file string) (string, error) {
	base, err := ModelCacheDir()
	if err != nil {
		return "", err
	}
	// Sanitize repo for use as dir name
	safeRepo := strings.ReplaceAll(repo, "/", "_")
	return filepath.Join(base, safeRepo, file), nil
}

// Download fetches url to destPath. Creates parent dirs. Uses ctx for timeout/cancel.
func Download(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "qmd/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// ResolveModel resolves model spec to a local GGUF path, downloading from Hugging Face if needed.
// spec can be:
//   - local path to a .gguf file (returned as-is; created if missing for later use)
//   - "repo:file.gguf" (Hugging Face; download to cache)
//   - "org/repo/file.gguf" (Hugging Face; last component is file)
func ResolveModel(ctx context.Context, spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	// Hugging Face: "repo:filename"
	if idx := strings.Index(spec, ":"); idx >= 0 {
		repo := strings.TrimPrefix(spec[:idx], "hf:")
		file := spec[idx+1:]
		if repo != "" && file != "" && strings.HasSuffix(file, ".gguf") {
			dest, err := LocalPath(repo, file)
			if err != nil {
				return "", err
			}
			if _, err := os.Stat(dest); err == nil {
				return dest, nil
			}
			url := ResolveURL(repo, DefaultRevision, file)
			if err := Download(ctx, url, dest); err != nil {
				return "", fmt.Errorf("download %s: %w", url, err)
			}
			return dest, nil
		}
	}
	// Hugging Face: "org/repo/filename.gguf" (no colon; path with .gguf)
	if strings.HasSuffix(spec, ".gguf") && strings.Count(spec, "/") >= 2 && !filepath.IsAbs(spec) && !strings.HasPrefix(spec, ".") {
		last := strings.LastIndex(spec, "/")
		repo := spec[:last]
		file := spec[last+1:]
		dest, err := LocalPath(repo, file)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(dest); err == nil {
			return dest, nil
		}
		url := ResolveURL(repo, DefaultRevision, file)
		if err := Download(ctx, url, dest); err != nil {
			return "", fmt.Errorf("download %s: %w", url, err)
		}
		return dest, nil
	}
	// Local path
	return spec, nil
}
