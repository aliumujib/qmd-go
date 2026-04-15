package store

import (
	"os"
	"testing"
	"time"
)

func TestDocumentsCRUD(t *testing.T) {
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

	content := "test content"
	hash := HashContent(content)
	now := time.Now().Truncate(time.Second)

	// Insert Content
	if err := s.InsertContent(hash, content, now); err != nil {
		t.Fatalf("InsertContent failed: %v", err)
	}

	// Insert Document
	if err := s.InsertDocument("testcol", "path/to/doc.md", "Test Title", hash, now, now); err != nil {
		t.Fatalf("InsertDocument failed: %v", err)
	}

	// Find Document
	doc, err := s.FindActiveDocument("testcol", "path/to/doc.md")
	if err != nil {
		t.Fatalf("FindActiveDocument failed: %v", err)
	}
	if doc.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got '%s'", doc.Title)
	}

	// Update Document Title
	if err := s.UpdateDocumentTitle(doc.ID, "New Title", now); err != nil {
		t.Fatalf("UpdateDocumentTitle failed: %v", err)
	}
	doc, _ = s.FindActiveDocument("testcol", "path/to/doc.md")
	if doc.Title != "New Title" {
		t.Errorf("Expected title 'New Title', got '%s'", doc.Title)
	}

	// Get Active Paths
	paths, err := s.GetActiveDocumentPaths("testcol")
	if err != nil {
		t.Fatalf("GetActiveDocumentPaths failed: %v", err)
	}
	if len(paths) != 1 || paths[0] != "path/to/doc.md" {
		t.Errorf("Unexpected paths: %v", paths)
	}

	// Deactivate
	if err := s.DeactivateDocument("testcol", "path/to/doc.md"); err != nil {
		t.Fatalf("DeactivateDocument failed: %v", err)
	}
	_, err = s.FindActiveDocument("testcol", "path/to/doc.md")
	if err == nil {
		t.Error("Expected error finding deactivated document")
	}

	// Cleanup
	affected, err := s.CleanupOrphanedContent()
	if err != nil {
		t.Fatalf("CleanupOrphanedContent failed: %v", err)
	}
	if affected != 1 {
		t.Errorf("Expected 1 orphaned content deleted, got %d", affected)
	}
}
