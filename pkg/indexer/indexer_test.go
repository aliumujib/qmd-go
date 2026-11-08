package indexer

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/aliumujib/qmd-go/pkg/config"
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

func TestIndexFilesWithPatternsExcludeAndMaxBytes(t *testing.T) {
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "notes", "venv"), 0o755))
	must(os.WriteFile(filepath.Join(root, "notes", "a.md"), []byte("# A"), 0o644))
	must(os.WriteFile(filepath.Join(root, "notes", "q.sql"), []byte("select 1"), 0o644))
	must(os.WriteFile(filepath.Join(root, "notes", "venv", "skip.md"), []byte("# venv"), 0o644))
	must(os.WriteFile(filepath.Join(root, "notes", "big.md"), make([]byte, 2048), 0o644))
	must(os.WriteFile(filepath.Join(root, "notes", "pic.png"), []byte{1, 2}, 0o644))

	s, err := store.NewStore(filepath.Join(t.TempDir(), "i.sqlite"))
	must(err)
	defer s.Close()

	opts := IndexOptions{Patterns: []string{"**/*.md", "**/*.sql"}, Exclude: []string{"**/venv/**"}, MaxBytes: 1024}
	must(IndexFilesWith(s, "col", root, opts))

	got, err := s.GetActiveDocumentPaths("col")
	must(err)
	sort.Strings(got)
	want := []string{"notes/a.md", "notes/q.sql"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("active = %v, want %v", got, want)
	}

	// A file that grows past the cap is deactivated on the next run.
	must(os.WriteFile(filepath.Join(root, "notes", "a.md"), make([]byte, 4096), 0o644))
	must(IndexFilesWith(s, "col", root, opts))
	got, _ = s.GetActiveDocumentPaths("col")
	if !reflect.DeepEqual(got, []string{"notes/q.sql"}) {
		t.Fatalf("after growth active = %v", got)
	}
}

func TestOptionsForFallsBackToPattern(t *testing.T) {
	o := OptionsFor(config.Collection{Pattern: "**/*.md"})
	if !reflect.DeepEqual(o.Patterns, []string{"**/*.md"}) || o.MaxBytes != 0 {
		t.Fatalf("got %+v", o)
	}
	o = OptionsFor(config.Collection{Pattern: "x", Patterns: []string{"a", "b"}, Exclude: []string{"c"}, MaxBytes: 9})
	if !reflect.DeepEqual(o.Patterns, []string{"a", "b"}) || o.Exclude[0] != "c" || o.MaxBytes != 9 {
		t.Fatalf("got %+v", o)
	}
}
