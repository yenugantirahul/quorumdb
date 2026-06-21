package storage

import (
	"fmt"
)

type MemoryStorage struct {
	data map[string]string
}

func NewMemoryStore() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]string),
	}
}

func (m *MemoryStorage) Set(key string, value string) {

	m.data[key] = value
	fmt.Println(m.data[key])
}

func (m *MemoryStorage) Get(key string) (string, bool) {
	value, ok := m.data[key]
	return value, ok
}

func (m *MemoryStorage) Delete(key string) {
	delete(m.data, key)
}
