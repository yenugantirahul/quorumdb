package handoff

import (
	"sync"

	"github.com/yenuganti/quorumdb/internal/model"
)

type Hint struct {
	NodeID string
	Key    string
	Record model.Record
}

type Manager struct {
	hints map[string][]Hint
	mu    sync.Mutex
}
