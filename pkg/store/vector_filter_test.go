package store

import (
	"path/filepath"
	"testing"
	"time"
)

// QA A-MCP-17: a collection filter on hybrid search must also filter the vector side.
func TestSearchVectorsBruteInFiltersCollection(t *testing.T) {
	s, err := NewStore(filepath.Join(t.TempDir(), "v.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnsureEmbeddingBlobTable(); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	add := func(col, path, body string, vec []float32) {
		h := HashContent(body)
		s.InsertContent(h, body, now)
		if err := s.InsertDocument(col, path, path, h, now, now); err != nil {
			t.Fatal(err)
		}
		if err := s.InsertEmbedding(h, 0, 0, vec, "m", now); err != nil {
			t.Fatal(err)
		}
	}
	add("research", "a.md", "panel scoring note", []float32{1, 0})
	add("claude-code", "s.md", "session about panel scoring", []float32{1, 0.01})

	all, err := s.SearchVectorsBrute([]float32{1, 0}, 10)
	if err != nil || len(all) != 2 {
		t.Fatalf("unfiltered = %v, %v", all, err)
	}
	got, err := s.SearchVectorsBruteIn([]float32{1, 0}, 10, "research")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Filepath != "qmd://research/a.md" {
		t.Fatalf("filtered = %+v", got)
	}
	if got, _ := s.SearchVectorsBruteIn([]float32{1, 0}, 10, "research' OR 1=1 --"); len(got) != 0 {
		t.Fatalf("injection string matched: %+v", got)
	}
}
