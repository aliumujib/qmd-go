/**
 * Evaluation Harness for QMD Search (Go port)
 *
 * Tests search quality with synthetic queries against known documents.
 * Uses a temporary index with test/eval-docs; mirrors test/eval-harness.ts.
 * Run: go test -v ./test/ -run EvalHarness
 */

package eval_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/aliumujib/qmd-go/pkg/indexer"
	"github.com/aliumujib/qmd-go/pkg/store"
)

const evalCollection = "eval-docs"

type difficulty string

const (
	difficultyEasy   difficulty = "easy"
	difficultyMedium difficulty = "medium"
	difficultyHard   difficulty = "hard"
)

type evalQuery struct {
	query       string
	expectedDoc string // partial match on filename
	difficulty  difficulty
	description string
}

var evalQueries = []evalQuery{
	// EASY: Exact keyword matches
	{"API versioning", "api-design", difficultyEasy, "Direct keyword match"},
	{"Series A fundraising", "fundraising", difficultyEasy, "Direct keyword match"},
	{"CAP theorem", "distributed-systems", difficultyEasy, "Direct keyword match"},
	{"overfitting machine learning", "machine-learning", difficultyEasy, "Direct keyword match"},
	{"remote work VPN", "remote-work", difficultyEasy, "Direct keyword match"},
	{"Project Phoenix retrospective", "product-launch", difficultyEasy, "Direct keyword match"},
	// MEDIUM: Semantic/conceptual queries
	{"how to structure REST endpoints", "api-design", difficultyMedium, "Conceptual - no exact match"},
	{"raising money for startup", "fundraising", difficultyMedium, "Conceptual - synonyms"},
	{"consistency vs availability tradeoffs", "distributed-systems", difficultyMedium, "Conceptual understanding"},
	{"how to prevent models from memorizing data", "machine-learning", difficultyMedium, "Conceptual - overfitting"},
	{"working from home guidelines", "remote-work", difficultyMedium, "Synonym match"},
	{"what went wrong with the launch", "product-launch", difficultyMedium, "Conceptual query"},
	// HARD: Vague, partial memory, indirect
	{"nouns not verbs", "api-design", difficultyHard, "Partial phrase recall"},
	{"Sequoia investor pitch", "fundraising", difficultyHard, "Indirect reference"},
	{"Raft algorithm leader election", "distributed-systems", difficultyHard, "Specific detail in long doc"},
	{"F1 score precision recall", "machine-learning", difficultyHard, "Technical detail"},
	{"quarterly team gathering travel", "remote-work", difficultyHard, "Specific policy detail"},
	{"beta program 47 bugs", "product-launch", difficultyHard, "Specific number recall"},
}

func findEvalDocsDir(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for _, base := range []string{filepath.Join(cwd, "test", "eval-docs"), filepath.Join(cwd, "eval-docs")} {
		if info, err := os.Stat(base); err == nil && info.IsDir() {
			return base
		}
	}
	t.Fatalf("eval-docs directory not found from %q; run tests from repo root", cwd)
	return ""
}

type difficultyStats struct {
	total, hit1, hit3, hit5 int
}

func (d *difficultyStats) add(firstHitRank int) {
	d.total++
	if firstHitRank == 1 {
		d.hit1++
	}
	if firstHitRank >= 1 && firstHitRank <= 3 {
		d.hit3++
	}
	if firstHitRank >= 1 && firstHitRank <= 5 {
		d.hit5++
	}
}

func firstMatchingRank(results []store.SearchResult, expectedDoc string) int {
	expectedLower := strings.ToLower(expectedDoc)
	for i, r := range results {
		fileLower := strings.ToLower(r.Filepath)
		if strings.Contains(fileLower, expectedLower) {
			return i + 1
		}
		displayLower := strings.ToLower(r.DisplayPath)
		if strings.Contains(displayLower, expectedLower) {
			return i + 1
		}
	}
	return -1
}

func TestEvalHarnessSearch(t *testing.T) {
	evalDocsDir := findEvalDocsDir(t)

	tmpFile, err := os.CreateTemp("", "qmd-eval-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	s, err := store.NewStore(dbPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such module: fts5") {
			t.Skip("sqlite3 built without FTS5; skip eval harness")
		}
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()

	err = indexer.IndexFiles(s, evalCollection, evalDocsDir, "**/*.md")
	if err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}

	stats := map[difficulty]*difficultyStats{
		difficultyEasy:   {},
		difficultyMedium: {},
		difficultyHard:   {},
	}

	t.Log("=== Evaluating SEARCH mode ===")
	for _, q := range evalQueries {
		results, err := s.SearchFTS(q.query, 5, "")
		if err != nil {
			t.Logf("SearchFTS %q: %v", q.query, err)
			stats[q.difficulty].add(-1)
			continue
		}
		rank := firstMatchingRank(results, q.expectedDoc)
		stats[q.difficulty].add(rank)

		status := "✗"
		if rank == 1 {
			status = "✓"
		} else if rank > 0 {
			status = "@" + strconv.Itoa(rank)
		}
		difficultyStr := string(q.difficulty)
		for len(difficultyStr) < 6 {
			difficultyStr += " "
		}
		for len(status) < 3 {
			status += " "
		}
		t.Logf("[%s] %s %q → %s", difficultyStr, status, q.query, q.description)
	}

	t.Log("--- Summary ---")
	for _, diff := range []difficulty{difficultyEasy, difficultyMedium, difficultyHard} {
		r := stats[diff]
		if r.total == 0 {
			continue
		}
		hit1Pct := 0
		if r.total > 0 {
			hit1Pct = r.hit1 * 100 / r.total
		}
		hit3Pct := 0
		if r.total > 0 {
			hit3Pct = r.hit3 * 100 / r.total
		}
		hit5Pct := 0
		if r.total > 0 {
			hit5Pct = r.hit5 * 100 / r.total
		}
		t.Logf("%-8s: Hit@1=%d%% Hit@3=%d%% Hit@5=%d%% (n=%d)", diff, hit1Pct, hit3Pct, hit5Pct, r.total)
	}
	total := len(evalQueries)
	totalHit1 := 0
	totalHit3 := 0
	for _, r := range stats {
		totalHit1 += r.hit1
		totalHit3 += r.hit3
	}
	overall1 := 0
	overall3 := 0
	if total > 0 {
		overall1 = totalHit1 * 100 / total
		overall3 = totalHit3 * 100 / total
	}
	t.Logf("Overall: Hit@1=%d%% Hit@3=%d%%", overall1, overall3)
}
