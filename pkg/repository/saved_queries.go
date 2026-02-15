package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	bolt "go.etcd.io/bbolt"
)

type SavedQuery struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
	Database    string `json:"database"`
	Tags        string `json:"tags"`
	Shared      bool   `json:"shared"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func (r *Repository) ListSavedQueries(_ context.Context, database string) ([]SavedQuery, error) {
	var result []SavedQuery
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSavedQueries)
		return b.ForEach(func(k, v []byte) error {
			var q SavedQuery
			if err := json.Unmarshal(v, &q); err != nil {
				return nil // skip corrupt entries
			}
			if database == "" || q.Database == database || q.Database == "" {
				result = append(result, q)
			}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list saved queries: %w", err)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt > result[j].UpdatedAt
	})
	if result == nil {
		result = []SavedQuery{}
	}
	return result, nil
}

func (r *Repository) GetSavedQuery(_ context.Context, id string) (*SavedQuery, error) {
	var q SavedQuery
	err := r.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketSavedQueries).Get([]byte(id))
		if v == nil {
			return fmt.Errorf("saved query not found: %s", id)
		}
		return json.Unmarshal(v, &q)
	})
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func (r *Repository) CreateSavedQuery(_ context.Context, q SavedQuery) (*SavedQuery, error) {
	q.ID = newID()
	now := nowUTC()
	q.CreatedAt = now
	q.UpdatedAt = now

	err := r.db.Update(func(tx *bolt.Tx) error {
		data, err := json.Marshal(q)
		if err != nil {
			return err
		}
		return tx.Bucket(bucketSavedQueries).Put([]byte(q.ID), data)
	})
	if err != nil {
		return nil, fmt.Errorf("create saved query: %w", err)
	}
	return &q, nil
}

func (r *Repository) UpdateSavedQuery(_ context.Context, q SavedQuery) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSavedQueries)
		v := b.Get([]byte(q.ID))
		if v == nil {
			return fmt.Errorf("saved query not found: %s", q.ID)
		}
		var existing SavedQuery
		if err := json.Unmarshal(v, &existing); err != nil {
			return err
		}
		existing.Title = q.Title
		existing.Description = q.Description
		existing.SQL = q.SQL
		existing.Database = q.Database
		existing.Tags = q.Tags
		existing.UpdatedAt = nowUTC()

		data, err := json.Marshal(existing)
		if err != nil {
			return err
		}
		return b.Put([]byte(q.ID), data)
	})
}

func (r *Repository) DeleteSavedQuery(_ context.Context, id string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSavedQueries).Delete([]byte(id))
	})
}

func (r *Repository) ListSharedQueries(_ context.Context) ([]SavedQuery, error) {
	var result []SavedQuery
	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSavedQueries)
		return b.ForEach(func(k, v []byte) error {
			var q SavedQuery
			if err := json.Unmarshal(v, &q); err != nil {
				return nil
			}
			if q.Shared {
				result = append(result, q)
			}
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("list shared queries: %w", err)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Title < result[j].Title
	})
	if result == nil {
		result = []SavedQuery{}
	}
	return result, nil
}

func (r *Repository) UpsertSharedQuery(_ context.Context, title, sql, description, database, tags string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSavedQueries)

		// Find existing by title + shared
		var existingID string
		b.ForEach(func(k, v []byte) error {
			var q SavedQuery
			if err := json.Unmarshal(v, &q); err != nil {
				return nil
			}
			if q.Shared && q.Title == title {
				existingID = q.ID
			}
			return nil
		})

		now := nowUTC()
		if existingID != "" {
			v := b.Get([]byte(existingID))
			var q SavedQuery
			if err := json.Unmarshal(v, &q); err != nil {
				return err
			}
			q.SQL = sql
			q.Description = description
			q.Database = database
			q.Tags = tags
			q.UpdatedAt = now
			data, err := json.Marshal(q)
			if err != nil {
				return err
			}
			return b.Put([]byte(existingID), data)
		}

		q := SavedQuery{
			ID:          newID(),
			Title:       title,
			Description: description,
			SQL:         sql,
			Database:    database,
			Tags:        tags,
			Shared:      true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		data, err := json.Marshal(q)
		if err != nil {
			return err
		}
		return b.Put([]byte(q.ID), data)
	})
}
