package repository

import bolt "go.etcd.io/bbolt"

var tabStateKey = []byte("state")

func (r *Repository) GetTabState() (string, error) {
	var data string
	err := r.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket(bucketTabs).Get(tabStateKey)
		if v != nil {
			data = string(v)
		}
		return nil
	})
	return data, err
}

func (r *Repository) SaveTabState(data string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketTabs).Put(tabStateKey, []byte(data))
	})
}
