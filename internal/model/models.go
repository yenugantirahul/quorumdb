package model

import (
	"github.com/yenuganti/quorumdb/internal/cluster"
)

type PutRequest struct {
	Value   string `json:"value"`
	Version uint64 `json:"version,omitempty"`
}
type Record struct {
	Value   string `json:"value"`
	Version uint64 `json:"version"`
}

type ReplicaResponse struct {
	Node   cluster.Node
	Record Record
}

type Resposnse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Error string `json:"error"`
}
