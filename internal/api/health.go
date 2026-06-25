package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yenuganti/quorumdb/internal/cluster"
)

type HealthManager struct {
	manager *cluster.Manager
	self    cluster.Node
}

type HealthResponse struct {
	Status  string `json:"status"`
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

func NewHealthManager(
	manager *cluster.Manager,
	self cluster.Node,
) *HealthManager {
	return &HealthManager{
		manager: manager,
		self:    self,
	}
}

func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "alive")
}
func (h *Handler) Health(w http.ResponseWriter,
	r *http.Request) {
	response := HealthResponse{
		Status:  "healthy",
		NodeID:  "node-1",
		Address: "localhost:8080",
	}
	json.NewEncoder(w).Encode(response)
}
