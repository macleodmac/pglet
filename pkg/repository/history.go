package repository

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type HistoryEntry struct {
	ID         int    `json:"id"`
	SQL        string `json:"sql"`
	Database   string `json:"database"`
	DurationMs int64  `json:"duration_ms"`
	RowCount   int    `json:"row_count"`
	Error      string `json:"error"`
	ExecutedAt string `json:"executed_at"`
}

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func (r *Repository) AddHistoryEntry(e HistoryEntry) error {
	e.ExecutedAt = nowUTC()
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketHistory)
		seq, err := b.NextSequence()
		if err != nil {
			return err
		}
		e.ID = int(seq)
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return b.Put(itob(seq), data)
	})
}

func (r *Repository) ListHistory(limit, offset int) ([]HistoryEntry, int, error) {
	var result []HistoryEntry
	total := 0

	err := r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketHistory)
		total = b.Stats().KeyN

		// Iterate in reverse (newest first) using a cursor
		c := b.Cursor()
		skipped := 0
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			if skipped < offset {
				skipped++
				continue
			}
			if len(result) >= limit {
				break
			}
			var e HistoryEntry
			if err := json.Unmarshal(v, &e); err != nil {
				continue
			}
			result = append(result, e)
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list history: %w", err)
	}
	if result == nil {
		result = []HistoryEntry{}
	}
	return result, total, nil
}

func (r *Repository) ClearHistory() error {
	return r.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(bucketHistory); err != nil {
			return err
		}
		_, err := tx.CreateBucket(bucketHistory)
		return err
	})
}
