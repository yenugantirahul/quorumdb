package version

import "sync/atomic"

type Manager struct {
	counter uint64
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Next() uint64 {
	return atomic.AddUint64(&m.counter, 1)
}
