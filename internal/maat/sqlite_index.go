package maat

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const sqliteIndexVersion = 1

type SQLiteIndexOptions struct {
	Store      string
	Path       string
	DisableFTS bool
}

type SQLiteIndexInfo struct {
	Path      string
	Documents int
	FTS       bool
}

type SQLiteIndex struct {
	db  *sql.DB
	fts bool
}

func SQLiteIndexPath(store string) string {
	return filepath.Join(store, ".maat", "index.sqlite")
}

func RebuildSQLiteIndex(store string) (SQLiteIndexInfo, error) {
	return RebuildSQLiteIndexWithOptions(SQLiteIndexOptions{Store: store})
}

func RebuildSQLiteIndexWithOptions(opts SQLiteIndexOptions) (SQLiteIndexInfo, error) {
	if strings.TrimSpace(opts.Store) == "" {
		return SQLiteIndexInfo{}, errors.New("store is required")
	}
	path := opts.Path
	if path == "" {
		path = SQLiteIndexPath(opts.Store)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return SQLiteIndexInfo{}, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return SQLiteIndexInfo{}, err
	}
	defer db.Close()

	if err := rebuildSQLiteSchema(db); err != nil {
		return SQLiteIndexInfo{}, err
	}
	fts, err := tryCreateFTS(db, opts.DisableFTS)
	if err != nil {
		return SQLiteIndexInfo{}, err
	}
	documents, err := collectDocuments(opts.Store)
	if err != nil {
		return SQLiteIndexInfo{}, err
	}
	if err := insertSQLiteDocuments(db, documents, fts); err != nil {
		return SQLiteIndexInfo{}, err
	}
	if err := writeSQLiteMetadata(db, fts); err != nil {
		return SQLiteIndexInfo{}, err
	}
	return SQLiteIndexInfo{Path: path, Documents: len(documents), FTS: fts}, nil
}

func OpenSQLiteIndex(path string) (*SQLiteIndex, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	fts, err := sqliteFTSEnabled(db)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteIndex{db: db, fts: fts}, nil
}

func (idx *SQLiteIndex) Close() error {
	if idx == nil || idx.db == nil {
		return nil
	}
	return idx.db.Close()
}

func (idx *SQLiteIndex) Search(query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if idx == nil || idx.db == nil {
		return nil, errors.New("sqlite index is not open")
	}
	if idx.fts {
		results, err := idx.searchFTS(query)
		if err == nil {
			return results, nil
		}
	}
	return idx.searchFallback(query)
}

func SearchSQLiteIndex(path, query string) ([]SearchResult, error) {
	idx, err := OpenSQLiteIndex(path)
	if err != nil {
		return nil, err
	}
	defer idx.Close()
	return idx.Search(query)
}

func rebuildSQLiteSchema(db *sql.DB) error {
	statements := []string{
		`PRAGMA journal_mode = WAL`,
		`DROP TABLE IF EXISTS search_documents_fts`,
		`DROP TABLE IF EXISTS search_documents`,
		`DROP TABLE IF EXISTS index_metadata`,
		`CREATE TABLE index_metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE search_documents (
			path TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			hash TEXT NOT NULL,
			size INTEGER NOT NULL,
			indexed_at TEXT NOT NULL
		)`,
		`CREATE INDEX search_documents_type_idx ON search_documents(type)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func tryCreateFTS(db *sql.DB, disabled bool) (bool, error) {
	if disabled {
		return false, nil
	}
	_, err := db.Exec(`CREATE VIRTUAL TABLE search_documents_fts USING fts5(
		path UNINDEXED,
		type UNINDEXED,
		title,
		content
	)`)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func insertSQLiteDocuments(db *sql.DB, documents []Document, fts bool) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertDocument, err := tx.Prepare(`INSERT INTO search_documents
		(path, type, title, content, hash, size, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insertDocument.Close()

	var insertFTS *sql.Stmt
	if fts {
		insertFTS, err = tx.Prepare(`INSERT INTO search_documents_fts
			(path, type, title, content)
			VALUES (?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer insertFTS.Close()
	}

	indexedAt := time.Now().UTC().Format(time.RFC3339)
	for _, document := range documents {
		hash := hashContent(document.Content)
		if _, err := insertDocument.Exec(document.Path, document.Type, document.Title, document.Content, hash, len(document.Content), indexedAt); err != nil {
			return err
		}
		if insertFTS != nil {
			if _, err := insertFTS.Exec(document.Path, document.Type, document.Title, document.Content); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func writeSQLiteMetadata(db *sql.DB, fts bool) error {
	ftsValue := "false"
	if fts {
		ftsValue = "true"
	}
	entries := map[string]string{
		"version":    fmt.Sprintf("%d", sqliteIndexVersion),
		"fts":        ftsValue,
		"rebuilt_at": time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range entries {
		if _, err := db.Exec(`INSERT INTO index_metadata (key, value) VALUES (?, ?)`, key, value); err != nil {
			return err
		}
	}
	return nil
}

func sqliteFTSEnabled(db *sql.DB) (bool, error) {
	var value string
	err := db.QueryRow(`SELECT value FROM index_metadata WHERE key = 'fts'`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

func (idx *SQLiteIndex) searchFTS(query string) ([]SearchResult, error) {
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return idx.searchFallback(query)
	}
	rows, err := idx.db.Query(`SELECT d.type, d.path, d.title, d.content
		FROM search_documents_fts f
		JOIN search_documents d ON d.path = f.path
		WHERE search_documents_fts MATCH ?
		ORDER BY bm25(search_documents_fts), d.path
		LIMIT 50`, ftsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchRows(rows, query)
}

func (idx *SQLiteIndex) searchFallback(query string) ([]SearchResult, error) {
	like := "%" + strings.ToLower(query) + "%"
	rows, err := idx.db.Query(`SELECT type, path, title, content
		FROM search_documents
		WHERE lower(path) LIKE ?
			OR lower(type) LIKE ?
			OR lower(title) LIKE ?
			OR lower(content) LIKE ?
		ORDER BY path
		LIMIT 50`, like, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchRows(rows, query)
}

func scanSearchRows(rows *sql.Rows, query string) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var content string
		if err := rows.Scan(&result.Type, &result.Path, &result.Title, &content); err != nil {
			return nil, err
		}
		result.Line, result.Excerpt = bestExcerpt(content, query)
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Path == results[j].Path {
			return results[i].Line < results[j].Line
		}
		return results[i].Path < results[j].Path
	})
	return results, nil
}

func bestExcerpt(content, query string) (int, string) {
	query = strings.ToLower(strings.TrimSpace(query))
	tokens := queryTokens(query)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		if query != "" && strings.Contains(lower, query) {
			return i + 1, strings.TrimSpace(line)
		}
	}
	for i, line := range lines {
		lower := strings.ToLower(line)
		for _, token := range tokens {
			if strings.Contains(lower, token) {
				return i + 1, strings.TrimSpace(line)
			}
		}
	}
	return 1, strings.TrimSpace(firstHeading(content))
}

func buildFTSQuery(query string) string {
	tokens := queryTokens(query)
	for i, token := range tokens {
		tokens[i] = token + "*"
	}
	return strings.Join(tokens, " AND ")
}

var queryTokenPattern = regexp.MustCompile(`[[:alnum:]_]+`)

func queryTokens(query string) []string {
	raw := queryTokenPattern.FindAllString(strings.ToLower(query), -1)
	tokens := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, token := range raw {
		token = strings.TrimSpace(token)
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		tokens = append(tokens, token)
	}
	return tokens
}

func hashContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
