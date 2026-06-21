package cluster

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
