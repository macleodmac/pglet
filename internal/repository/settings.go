package repository

import (
	"crypto/rand"
	"encoding/hex"

	bolt "go.etcd.io/bbolt"
)

var (
	authTokenKey        = []byte("auth_token")
	savedConnectionsKey = []byte("saved_connections")
)

// AuthToken returns the token the browser must present, creating it on first
// use. It is stable across restarts so bookmarked sessions keep working.
func (r *Repository) AuthToken() (string, error) {
	var token string
	err := r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketSettings)
		if v := b.Get(authTokenKey); v != nil {
			token = string(v)
			return nil
		}
		buf := make([]byte, 24)
		if _, err := rand.Read(buf); err != nil {
			return err
		}
		token = hex.EncodeToString(buf)
		return b.Put(authTokenKey, []byte(token))
	})
	return token, err
}

// SavedConnections returns the saved connection list as stored JSON.
func (r *Repository) SavedConnections() (string, error) {
	var data string
	err := r.db.View(func(tx *bolt.Tx) error {
		data = string(tx.Bucket(bucketSettings).Get(savedConnectionsKey))
		return nil
	})
	return data, err
}

func (r *Repository) SaveConnections(data string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketSettings).Put(savedConnectionsKey, []byte(data))
	})
}
