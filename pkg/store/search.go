package store

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

type SearchResult struct {
	Filepath       string
	DisplayPath    string
	Title          string
	Body           string
	Hash           string
	Score          float64
	Source         string
	CollectionName string
}

func SanitizeFTS5Term(term string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9']`)
	return strings.ToLower(reg.ReplaceAllString(term, ""))
}

func BuildFTS5Query(query string) string {
	terms := strings.Fields(query)
	var validTerms []string
	for _, t := range terms {
		sanitized := SanitizeFTS5Term(t)
		if len(sanitized) > 0 {
			validTerms = append(validTerms, fmt.Sprintf(`"%s"*`, sanitized))
		}
	}
	if len(validTerms) == 0 {
		return ""
	}
	return strings.Join(validTerms, " AND ")
}

func (s *Store) SearchFTS(query string, limit int, collectionFilter string) ([]SearchResult, error) {
	ftsQuery := BuildFTS5Query(query)
	if ftsQuery == "" {
		return []SearchResult{}, nil
	}

	sql := `
		SELECT
			'qmd://' || d.collection || '/' || d.path as filepath,
			d.collection || '/' || d.path as display_path,
			d.title,
			content.doc as body,
			d.hash,
			bm25(documents_fts, 10.0, 1.0) as bm25_score,
			d.collection
		FROM documents_fts f
		JOIN documents d ON d.id = f.rowid
		JOIN content ON content.hash = d.hash
		WHERE documents_fts MATCH ? AND d.active = 1
	`
	args := []interface{}{ftsQuery}
	if collectionFilter != "" {
		sql += ` AND d.collection = ?`
		args = append(args, collectionFilter)
	}
	sql += ` ORDER BY bm25_score ASC LIMIT ?`
	args = append(args, limit)

	rows, err := s.DB.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var bm25Score float64
		if err := rows.Scan(&r.Filepath, &r.DisplayPath, &r.Title, &r.Body, &r.Hash, &bm25Score, &r.CollectionName); err != nil {
			return nil, err
		}
		// Normalize BM25 (negative, lower is better)
		// Map to 0-1 where higher is better using sigmoid-ish logic from original
		absScore := math.Abs(bm25Score)
		r.Score = 1.0 / (1.0 + math.Exp(-(absScore-5.0)/3.0))
		r.Source = "fts"
		results = append(results, r)
	}
	return results, nil
}
