package api

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status  string `json:"status"`
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
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
