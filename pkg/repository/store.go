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
	bucketTabs         = []byte("tabs")
)

type Repository struct {
	db *bolt.DB
}

func Open(path string) (*Repository, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create repository dir: %w", err)
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}

	// Ensure buckets exist
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketSavedQueries, bucketHistory, bucketTabs} {
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
