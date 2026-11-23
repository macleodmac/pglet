package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketSavedQueries = []byte("saved_queries")
	bucketHistory      = []byte("history")
	bucketTabs         = []byte("tabs")
	bucketSettings     = []byte("settings")
)

type Repository struct {
	db *bolt.DB
	// sharedDir holds the shared query files; empty when there is none.
	sharedDir string
}

// Open opens the store at path. sharedDir is the directory of shared .sql
// query files, or "" for none.
func Open(path, sharedDir string) (*Repository, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create repository dir: %w", err)
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if errors.Is(err, bolt.ErrTimeout) {
		return nil, fmt.Errorf("%s is locked by another running pglet; stop it or use --store-dir", path)
	}
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}

	// Ensure buckets exist
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketSavedQueries, bucketHistory, bucketTabs, bucketSettings} {
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

	r := &Repository{db: db, sharedDir: sharedDir}
	if err := r.dropImportedSharedQueries(); err != nil {
		db.Close()
		return nil, err
	}
	return r, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}
