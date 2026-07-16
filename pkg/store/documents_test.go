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

func TestRecentDocuments(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "qmd-recent-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	s, err := NewStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	base := time.Date(2026, 7, 16, 5, 0, 0, 0, time.UTC)
	seed := func(path, body string, mod time.Time) {
		h := HashContent(body)
		if err := s.InsertContent(h, body, mod); err != nil {
			t.Fatal(err)
		}
		if err := s.InsertDocument("claude-code", path, "T", h, mod, mod); err != nil {
			t.Fatal(err)
		}
	}
	// two docs in project A (distinct mtimes) + one in project B
	seed("-proj-A/old.md", "old body", base.Add(-2*time.Hour))
	seed("-proj-A/new.md", "new body", base)
	seed("-proj-B/other.md", "other body", base.Add(-time.Hour))

	// filter to project A → only its 2 docs, newest first, with body joined
	got, err := s.RecentDocuments("", "-proj-A", 10)
	if err != nil {
		t.Fatalf("RecentDocuments: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 project-A docs, got %d: %+v", len(got), got)
	}
	if got[0].Path != "-proj-A/new.md" || got[1].Path != "-proj-A/old.md" {
		t.Fatalf("want newest-first, got %s then %s", got[0].Path, got[1].Path)
	}
	if got[0].Body != "new body" {
		t.Fatalf("body not joined: %q", got[0].Body)
	}

	// limit caps the result
	if lim, _ := s.RecentDocuments("", "-proj-A", 1); len(lim) != 1 || lim[0].Path != "-proj-A/new.md" {
		t.Fatalf("limit=1 should return only the newest, got %+v", lim)
	}

	// empty filter lists across all projects (3), still newest-first
	all, _ := s.RecentDocuments("", "", 10)
	if len(all) != 3 || all[0].Path != "-proj-A/new.md" {
		t.Fatalf("want 3 across all, newest first, got %d", len(all))
	}

	// LIKE metacharacters in the filter are matched literally, not as wildcards
	if none, _ := s.RecentDocuments("", "proj-%", 10); len(none) != 0 {
		t.Fatalf("LIKE wildcard must be escaped; want 0, got %d", len(none))
	}
}
