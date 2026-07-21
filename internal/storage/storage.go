package storage

import "github.com/yenuganti/quorumdb/internal/model"

type Store interface {
	Put(key string, record model.Record) error
	Get(key string) (model.Record, error)
	Delete(key string) error
}
