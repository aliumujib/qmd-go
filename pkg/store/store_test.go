package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

// A second writer waits for the first instead of failing with "database is locked".
func TestSecondWriterWaitsForBusyLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.sqlite")
	a, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	tx, err := a.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(`INSERT INTO content (hash, doc, created_at) VALUES ('a', 'a', 'now')`); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- b.InsertContent("b", "b", time.Now())
	}()
	time.Sleep(200 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second writer: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("second writer never finished")
	}
}
