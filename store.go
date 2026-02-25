package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type store struct {
	db *sql.DB
}

type searchResult struct {
	title   string
	snippet string
	score   float64
}

type page struct {
	title      string
	body       string
	source     string
	revision   string
	categories string
}

func createStore(path string) (*store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	_, err = db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS pages USING fts5(
		title, body,
		source UNINDEXED, revision UNINDEXED, categories UNINDEXED,
		tokenize = 'porter unicode61'
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create fts5 table: %w", err)
	}
	return &store{db: db}, nil
}

func openStore(path string) (*store, error) {
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &store{db: db}, nil
}

func (s *store) close() error { return s.db.Close() }

func (s *store) pageCount() (int, error) {
	var n int
	err := s.db.QueryRow("SELECT count(*) FROM pages").Scan(&n)
	return n, err
}

func (s *store) beginTx() (*sql.Tx, error) { return s.db.Begin() }

func (s *store) insertTx(tx *sql.Tx, title, body, source, revision, categories string) error {
	_, err := tx.Exec(
		"INSERT INTO pages(title, body, source, revision, categories) VALUES(?, ?, ?, ?, ?)",
		title, body, source, revision, categories,
	)
	return err
}

func (s *store) search(query string, limit int) ([]searchResult, error) {
	fts := sanitizeFTS(query)
	if fts == "" {
		return nil, nil
	}
	rows, err := s.db.Query(
		`SELECT title, snippet(pages, 1, '', '', '...', 40), bm25(pages, 5.0, 1.0)
		 FROM pages WHERE pages MATCH ? ORDER BY rank LIMIT ?`, fts, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()
	var results []searchResult
	for rows.Next() {
		var r searchResult
		if err := rows.Scan(&r.title, &r.snippet, &r.score); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (s *store) readPage(query string) (*page, error) {
	if p, err := s.queryPage(query); err == nil {
		return p, nil
	}
	normalized := strings.TrimSuffix(query, ".md")
	normalized = strings.ReplaceAll(normalized, "_", " ")
	if p, err := s.queryPage(normalized); err == nil {
		return p, nil
	}
	noExt := strings.TrimSuffix(query, ".md")
	if noExt != normalized {
		if p, err := s.queryPage(noExt); err == nil {
			return p, nil
		}
	}
	return nil, fmt.Errorf("page %q not found", query)
}

func (s *store) queryPage(title string) (*page, error) {
	p := &page{}
	err := s.db.QueryRow(
		"SELECT title, body, source, revision, categories FROM pages WHERE title = ? COLLATE NOCASE", title,
	).Scan(&p.title, &p.body, &p.source, &p.revision, &p.categories)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func sanitizeFTS(query string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	return strings.Join(quoted, " ")
}
