package storage

import (
	"encoding/json"

	"github.com/dgraph-io/badger/v4"
	"github.com/yenuganti/quorumdb/internal/model"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &BadgerStore{
		db: db,
	}, nil
}

// SET the key -> Value in the Disk
func (b *BadgerStore) Put(key string, record model.Record) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(data))
	})
}

// Gets the Key - Value from the Disk
func (b *BadgerStore) Get(key string) (model.Record, error) {
	var record model.Record

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &record)
		})
	})

	return record, err
}

// Deletes the key-value pair stored on the disk
func (b *BadgerStore) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
