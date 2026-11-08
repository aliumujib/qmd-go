package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/aliumujib/qmd-go/pkg/config"
	"github.com/aliumujib/qmd-go/pkg/store"
	"github.com/bmatcuk/doublestar/v4"
)

// IndexOptions selects the files a collection indexes.
type IndexOptions struct {
	Patterns []string // doublestar globs relative to the root; union
	Exclude  []string // doublestar globs; a match skips the file
	MaxBytes int64    // files larger than this are skipped (and deactivated if indexed); 0 = no cap
}

// OptionsFor turns a saved collection into index options.
func OptionsFor(c config.Collection) IndexOptions {
	pats := c.Patterns
	if len(pats) == 0 {
		pats = []string{c.Pattern}
	}
	return IndexOptions{Patterns: pats, Exclude: c.Exclude, MaxBytes: c.MaxBytes}
}

func IndexFiles(s *store.Store, collectionName, rootPath, pattern string) error {
	return IndexFilesWith(s, collectionName, rootPath, IndexOptions{Patterns: []string{pattern}})
}

// IndexFilesWith indexes the union of o.Patterns under rootPath, minus excluded and over-cap files.
// Files that were indexed before and no longer qualify are deactivated.
func IndexFilesWith(s *store.Store, collectionName, rootPath string, o IndexOptions) error {
	fsys := os.DirFS(rootPath)

	now := time.Now()

	seen := map[string]bool{}
	var files []string
	for _, p := range o.Patterns {
		matches, err := doublestar.Glob(fsys, p)
		if err != nil {
			return fmt.Errorf("pattern %q: %w", p, err)
		}
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				files = append(files, m)
			}
		}
	}
	sort.Strings(files)
	excluded := func(rel string) bool {
		for _, ex := range o.Exclude {
			if ok, _ := doublestar.Match(ex, rel); ok {
				return true
			}
		}
		return false
	}

	indexedCount := 0
	updatedCount := 0
	seenPaths := make(map[string]bool)

	for _, relPath := range files {
		if excluded(relPath) {
			continue
		}
		fullPath := filepath.Join(rootPath, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error stating file %s: %v\n", fullPath, err)
			continue
		}
		if info.IsDir() || (o.MaxBytes > 0 && info.Size() > o.MaxBytes) {
			continue
		}

		seenPaths[relPath] = true

		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", fullPath, err)
			continue
		}
		content := string(contentBytes)
		hash := store.HashContent(content)
		title := filepath.Base(relPath) // Simplified title extraction

		// Check if exists
		doc, err := s.FindActiveDocument(collectionName, relPath)
		if err == nil {
			// Update if changed
			if doc.Hash != hash {
				if err := s.InsertContent(hash, content, now); err != nil {
					fmt.Fprintf(os.Stderr, "Error inserting content for %s: %v\n", relPath, err)
					continue
				}
				if err := s.UpdateDocument(doc.ID, title, hash, now); err != nil {
					fmt.Fprintf(os.Stderr, "Error updating document %s: %v\n", relPath, err)
					continue
				}
				updatedCount++
			}
		} else {
			// Insert new
			if err := s.InsertContent(hash, content, now); err != nil {
				fmt.Fprintf(os.Stderr, "Error inserting content for %s: %v\n", relPath, err)
				continue
			}
			if err := s.InsertDocument(collectionName, relPath, title, hash, info.ModTime(), now); err != nil {
				fmt.Fprintf(os.Stderr, "Error inserting document %s: %v\n", relPath, err)
				continue
			}
			indexedCount++
		}
	}

	// Handle deletions
	activePaths, err := s.GetActiveDocumentPaths(collectionName)
	removedCount := 0
	if err == nil {
		for _, path := range activePaths {
			if !seenPaths[path] {
				if err := s.DeactivateDocument(collectionName, path); err != nil {
					fmt.Fprintf(os.Stderr, "Error deactivating document %s: %v\n", path, err)
					continue
				}
				removedCount++
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error getting active documents: %v\n", err)
	}

	// Cleanup orphans
	if _, err := s.CleanupOrphanedContent(); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning up orphans: %v\n", err)
	}

	// Progress goes to stderr: trace serves MCP over stdout.
	fmt.Fprintf(os.Stderr, "Collection '%s': Indexed %d new, Updated %d, Removed %d.\n", collectionName, indexedCount, updatedCount, removedCount)
	return nil
}
