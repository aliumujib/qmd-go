package indexer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aliumujib/qmd-go/pkg/store"
)

func TestIndexFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "qmd-index-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create dummy file
	os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("# Hello World"), 0644)

	// Create store
	tmpDb, err := os.CreateTemp("", "qmd-db-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpDb.Close()
	defer os.Remove(tmpDb.Name())

	s, err := store.NewStore(tmpDb.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := IndexFiles(s, "testcol", tmpDir, "*.md"); err != nil {
		t.Fatalf("IndexFiles failed: %v", err)
	}

	// Verify
	doc, err := s.FindActiveDocument("testcol", "test.md")
	if err != nil {
		t.Fatalf("Document not found: %v", err)
	}
	if doc.Title != "test.md" {
		t.Errorf("Expected title 'test.md', got '%s'", doc.Title)
	}

	// Update file
	os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte("# Hello World Updated"), 0644)

	// Re-index
	time.Sleep(10 * time.Millisecond)
	if err := IndexFiles(s, "testcol", tmpDir, "*.md"); err != nil {
		t.Fatalf("IndexFiles failed: %v", err)
	}

	// Verify update
	doc2, _ := s.FindActiveDocument("testcol", "test.md")
	if doc2.Hash == doc.Hash {
		t.Error("Hash should have changed")
	}

	// Delete file
	os.Remove(filepath.Join(tmpDir, "test.md"))

	// Re-index
	time.Sleep(10 * time.Millisecond)
	if err := IndexFiles(s, "testcol", tmpDir, "*.md"); err != nil {
		t.Fatalf("IndexFiles failed: %v", err)
	}

	// Verify deletion
	_, err = s.FindActiveDocument("testcol", "test.md")
	if err == nil {
		t.Error("Document should be deactivated")
	}
}
