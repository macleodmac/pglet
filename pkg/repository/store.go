package repository

import (
	"fmt"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketSavedQueries = []byte("saved_queries")
	bucketHistory      = []byte("history")
	bucketSettings     = []byte("settings")
)

type Repository struct {
	db *bolt.DB
}

func Open(path string) (*Repository, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create repository dir: %w", err)
	}

	// If an old sqlite DB exists, back it up
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		header := make([]byte, 16)
		if f, err := os.Open(path); err == nil {
			f.Read(header)
			f.Close()
			if string(header[:6]) == "SQLite" {
				backup := path + ".sqlite.bak"
				os.Rename(path, backup)
			}
		}
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}

	// Ensure buckets exist
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketSavedQueries, bucketHistory, bucketSettings} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("init buckets: %w", err)
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}
