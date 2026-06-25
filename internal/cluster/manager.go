package cluster

import (
	"fmt"
	"net/http"
	"time"
)

type Manager struct {
	nodes map[string]Node
}

func NewManager() *Manager {
	return &Manager{
		nodes: make(map[string]Node),
	}
}

func (m *Manager) AddNode(node Node) {
	m.nodes[node.ID] = node
}

func (m *Manager) GetNode(id string) (Node, bool) {
	node, ok := m.nodes[id]
	return node, ok
}

func (m *Manager) GetAllNodes() []Node {
	nodes := make([]Node, 0)

	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

func (m *Manager) IsAlive(id string) bool {
	node, ok := m.nodes[id]
	if !ok {
		return false
	}

	return node.Alive
}
func (m *Manager) SetAlive(id string, alive bool) {
	node, ok := m.nodes[id]
	if !ok {
		return
	}

	node.Alive = alive
	m.nodes[id] = node
}

type HealthManager struct {
	manager *Manager
	self    Node
}

func NewHealthManager(
	manager *Manager,
	self Node,
) *HealthManager {
	return &HealthManager{
		manager: manager,
		self:    self,
	}
}

func (h *HealthManager) Start() {

	// Create a ticker that ticks every 5 seconds
	ticker := time.NewTicker(5 * time.Second)

	// Run in the background
	go func() {

		// Every time the ticker fires, this loop executes once.
		for range ticker.C {

			// Get every node in the cluster
			nodes := h.manager.GetAllNodes()

			// Check every node
			for _, node := range nodes {

				// Avoid checking the current node
				if node.ID == h.self.ID {
					continue
				}

				url := fmt.Sprintf(
					"http://localhost:%s/health",
					node.PORT,
				)

				resp, err := http.Get(url)

				// Server unreachable
				if err != nil {
					h.manager.SetAlive(node.ID, false)
					fmt.Println(node.ID, "DOWN")
					continue
				}

				// Close connection
				resp.Body.Close()

				// Server responded
				if resp.StatusCode == http.StatusOK {
					h.manager.SetAlive(node.ID, true)
					fmt.Println(node.ID, "UP")
				} else {
					h.manager.SetAlive(node.ID, false)
					fmt.Println(node.ID, "DOWN")
				}
			}
		}
	}()
}
