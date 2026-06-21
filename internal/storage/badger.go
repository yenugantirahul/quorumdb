package storage

import (
	"github.com/dgraph-io/badger/v4"
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
func (b *BadgerStore) Put(key string, value string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), []byte(value))
	})
}

// Gets the Key - Value from the Disk

func (b *BadgerStore) Get(key string) (string, error) {
	var result string

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			result = string(val)
			return nil
		})
	})

	return result, err
}

// Deletes the key-value pair stored on the disk
func (b *BadgerStore) Delete(key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
