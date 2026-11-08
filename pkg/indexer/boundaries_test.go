package indexer

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aliumujib/qmd-go/pkg/store"
)

// QA A-IDX-05: include/exclude/size-cap boundaries, symlinks, empty, unreadable and invalid UTF-8 files.
func TestIndexFilesWithBoundaries(t *testing.T) {
	root := t.TempDir()
	const capBytes = 64
	write := func(rel string, data []byte, mode os.FileMode) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, mode); err != nil {
			t.Fatal(err)
		}
	}
	fill := func(n int) []byte { return []byte(strings.Repeat("x", n)) }
	write("a.md", fill(capBytes), 0o644)   // exactly the cap: indexed (cap is inclusive)
	write("b.md", fill(capBytes+1), 0o644) // one byte over: skipped
	write("c.MD", []byte("upper"), 0o644)  // globs are case-sensitive: skipped
	write("d.md.bak", []byte("bak"), 0o644)
	write(".hidden/e.md", []byte("hidden"), 0o644) // dot dirs follow the glob: `**` matches them
	write("node_modules/f.md", []byte("vendored"), 0o644)
	write("x/venv-old/g.md", []byte("venv"), 0o644)
	write("h.md", nil, 0o644)                           // empty: skipped
	write("i.md", []byte("caf\xe9 \xff\xfe ok"), 0o644) // invalid UTF-8: indexed, no panic
	write("j.md", []byte("locked"), 0o000)              // unreadable: skipped
	write("sub/k.md", []byte("kept"), 0o644)
	if err := os.Symlink(".", filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("root:x:0:0"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "p.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(root, "outdir")); err != nil {
		t.Fatal(err)
	}

	s, err := store.NewStore(filepath.Join(t.TempDir(), "i.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	opts := IndexOptions{
		Patterns: []string{"**/*.md", "**/*.txt"},
		Exclude:  []string{"**/node_modules/**", "**/*venv*/**"},
		MaxBytes: capBytes,
	}
	if err := IndexFilesWith(s, "col", root, opts); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetActiveDocumentPaths("col")
	sort.Strings(got)
	want := []string{".hidden/e.md", "a.md", "i.md", "sub/k.md"}
	if os.Geteuid() == 0 { // root reads a 0-permission file
		want = []string{".hidden/e.md", "a.md", "i.md", "j.md", "sub/k.md"}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("active = %v, want %v", got, want)
	}
	res, err := s.SearchFTS("root", 10, "col")
	if err != nil || len(res) != 0 {
		t.Fatalf("a symlink out of the tree was followed: %v %v", res, err)
	}
}

// A file that stops qualifying and later qualifies again is re-activated, even when its old content
// is still held by another document (the inactive row would otherwise hit UNIQUE(collection, path)).
func TestIndexFilesWithReactivatesReturningFile(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.md"), []byte("same words"), 0o644)
	os.WriteFile(filepath.Join(root, "twin.md"), []byte("same words"), 0o644)
	s, err := store.NewStore(filepath.Join(t.TempDir(), "i.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	opts := IndexOptions{Patterns: []string{"**/*.md"}}
	if err := IndexFilesWith(s, "col", root, opts); err != nil {
		t.Fatal(err)
	}
	if err := IndexFilesWith(s, "col", root, IndexOptions{Patterns: []string{"**/*.md"}, Exclude: []string{"a.md"}}); err != nil {
		t.Fatal(err)
	}
	if err := IndexFilesWith(s, "col", root, opts); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetActiveDocumentPaths("col")
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"a.md", "twin.md"}) {
		t.Fatalf("active = %v", got)
	}
}

// QA A-IDX-06: files that stop qualifying are deactivated; untouched files keep their embeddings.
func TestIndexFilesWithKeepsEmbeddingsOfUntouchedFiles(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "node_modules"), 0o755)
	os.WriteFile(filepath.Join(root, "a.md"), []byte("alpha"), 0o644)
	os.WriteFile(filepath.Join(root, "keep.md"), []byte("keep me"), 0o644)
	os.WriteFile(filepath.Join(root, "node_modules", "n.md"), []byte("vendored"), 0o644)
	s, err := store.NewStore(filepath.Join(t.TempDir(), "i.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.EnsureEmbeddingBlobTable(); err != nil {
		t.Fatal(err)
	}
	opts := IndexOptions{Patterns: []string{"**/*.md"}, MaxBytes: 32}
	if err := IndexFilesWith(s, "col", root, opts); err != nil {
		t.Fatal(err)
	}
	pending, _ := s.GetHashesForEmbedding()
	for _, p := range pending {
		if err := s.InsertEmbedding(p.Hash, 0, 0, []float32{1, 0}, "m", time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, "a.md"), []byte(strings.Repeat("grown ", 20)), 0o644)
	opts.Exclude = []string{"**/node_modules/**"}
	if err := IndexFilesWith(s, "col", root, opts); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetActiveDocumentPaths("col")
	if !reflect.DeepEqual(got, []string{"keep.md"}) {
		t.Fatalf("active = %v, want [keep.md]", got)
	}
	if n, _ := s.GetHashesNeedingEmbeddingCount(); n != 0 {
		t.Fatalf("%d docs need embedding again; untouched files must keep theirs", n)
	}
	if res, _ := s.SearchFTS("vendored", 5, "col"); len(res) != 0 {
		t.Fatalf("excluded file still searchable: %v", res)
	}
}

// IndexOptions.Transform rewrites content before it is hashed and stored (trace redacts secrets).
func TestIndexFilesWithTransform(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "a.md"), []byte("token sbp_0123456789abcdef here"), 0o644)
	s, err := store.NewStore(filepath.Join(t.TempDir(), "i.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	opts := IndexOptions{Patterns: []string{"*.md"}, Transform: func(c string) string {
		return strings.ReplaceAll(c, "sbp_0123456789abcdef", "[redacted]")
	}}
	if err := IndexFilesWith(s, "col", root, opts); err != nil {
		t.Fatal(err)
	}
	body, err := s.GetDocumentBody("col", "a.md", 0, 0)
	if err != nil || strings.Contains(body, "sbp_") {
		t.Fatalf("body = %q, %v", body, err)
	}
}
