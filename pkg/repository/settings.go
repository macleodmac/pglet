package repository

import (
	"context"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

func (r *Repository) GetSetting(_ context.Context, key string) (string, error) {
	var val string
	err := r.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketSettings).Get([]byte(key))
		if v == nil {
			return fmt.Errorf("setting not found: %s", key)
		}
		val = string(v)
		return nil
	})
	return val, err
}

func (r *Repository) SetSetting(_ context.Context, key, value string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSettings).Put([]byte(key), []byte(value))
	})
}

func (r *Repository) GetAllSettings(_ context.Context) (map[string]string, error) {
	result := make(map[string]string)
	err := r.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSettings).ForEach(func(k, v []byte) error {
			result[string(k)] = string(v)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
