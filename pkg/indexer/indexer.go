package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aliumujib/qmd-go/pkg/store"
	"github.com/bmatcuk/doublestar/v4"
)

func IndexFiles(s *store.Store, collectionName, rootPath, pattern string) error {
	fsys := os.DirFS(rootPath)

	now := time.Now()

	// Use doublestar to find files
	files, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return err
	}

	indexedCount := 0
	updatedCount := 0
	seenPaths := make(map[string]bool)

	for _, relPath := range files {
		fullPath := filepath.Join(rootPath, relPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error stating file %s: %v\n", fullPath, err)
			continue
		}
		if info.IsDir() {
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

	fmt.Printf("Collection '%s': Indexed %d new, Updated %d, Removed %d.\n", collectionName, indexedCount, updatedCount, removedCount)
	return nil
}
