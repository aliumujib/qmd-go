package store

import (
	"os"
	"testing"
	"time"
)

func TestSearchFTS(t *testing.T) {
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

	now := time.Now()
	content := "This is a test document about bananas."
	hash := HashContent(content)
	s.InsertContent(hash, content, now)
	s.InsertDocument("testcol", "banana.md", "Banana Doc", hash, now, now)

	content2 := "This is another document about apples."
	hash2 := HashContent(content2)
	s.InsertContent(hash2, content2, now)
	s.InsertDocument("testcol", "apple.md", "Apple Doc", hash2, now, now)

	// Search for "banana"
	results, err := s.SearchFTS("banana", 10, "")
	if err != nil {
		t.Fatalf("SearchFTS failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Banana Doc" {
		t.Errorf("Expected 'Banana Doc', got '%s'", results[0].Title)
	}

	// Search for "test"
	results, err = s.SearchFTS("test", 10, "")
	if err != nil {
		t.Fatalf("SearchFTS failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}
