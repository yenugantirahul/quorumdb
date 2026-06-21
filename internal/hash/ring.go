package hash

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/yenuganti/quorumdb/internal/cluster"
)

type HashRing struct {
	nodeHashes   map[uint64]cluster.Node
	sortedHashes []uint64
}

func NewHashRing() *HashRing {
	return &HashRing{
		nodeHashes:   make(map[uint64]cluster.Node),
		sortedHashes: make([]uint64, 0),
	}
}

func (h *HashRing) hash(key string) uint64 {
	hash := sha1.Sum([]byte(key))
	return binary.BigEndian.Uint64(hash[:8])
}

func (h *HashRing) AddNode(node cluster.Node) {

	for i := 0; i < 64; i++ {
		virtualNode := fmt.Sprintf("%s#%d", node.ID, i)
		hashedValue := h.hash(virtualNode)
		h.nodeHashes[hashedValue] = node
		h.sortedHashes = append(h.sortedHashes, hashedValue)
	}

	sort.Slice(h.sortedHashes, func(i, j int) bool {
		return h.sortedHashes[i] < h.sortedHashes[j]
	})
}

func (h *HashRing) GetNode(key string) []cluster.Node {
	nodes := make([]cluster.Node, 0, 3)
	hashedValue := h.hash(key)
	for _, val := range h.sortedHashes {
		if val >= hashedValue && len(nodes) < 3 {
			nodes = append(nodes, h.nodeHashes[val])
		} else if len(nodes) == 3 {
			break
		}
	}
	return nodes
}
