package store

import (
	"os"
	"testing"
)

func TestNewStore(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "qmd-test-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	s, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer s.Close()

	// Check if tables exist
	tables := []string{"content", "documents", "llm_cache", "content_vectors", "documents_fts"}
	for _, table := range tables {
		var name string
		err := s.DB.QueryRow("SELECT name FROM sqlite_master WHERE name = ?", table).Scan(&name)
		if err != nil {
			t.Errorf("Table %s not found: %v", table, err)
		}
	}
}
