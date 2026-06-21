package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yenuganti/quorumdb/internal/cluster"
	"github.com/yenuganti/quorumdb/internal/hash"
	"github.com/yenuganti/quorumdb/internal/model"
	"github.com/yenuganti/quorumdb/internal/storage"
)

// Handler Struct contains store, pointer
//
//	to hash-ring and the current nodes data as self
type Handler struct {
	store storage.Store
	ring  *hash.HashRing
	self  cluster.Node
}

func NewHandler(
	store storage.Store,
	ring *hash.HashRing,
	self cluster.Node,
) *Handler {
	return &Handler{
		store: store,
		ring:  ring,
		self:  self,
	}
}

// This function handles the reuqests
func (h *Handler) HandleKey(w http.ResponseWriter, r *http.Request) {
	// Defines the type of method as GET, PUT, PATCH, DELETE
	requestMethod := r.Method

	key := strings.TrimPrefix(
		r.URL.Path,
		"/key/",
	)

	// Hashes the current key which tells in which node the data should be stored
	owner := h.ring.GetNode(key)

	if owner[0].ID == h.self.ID {
		fmt.Fprintln(w, "Request received with method: "+requestMethod)

		switch r.Method {
		case "GET":
			res, err := h.store.Get(key)
			if err != nil {
				fmt.Fprintln(w, "Value Doesn't exist")
				return
			}
			fmt.Fprintln(w, res)
		case "PUT":
			var req model.PutRequest

			bodyBytes, err := io.ReadAll(r.Body)

			if err != nil {
				fmt.Fprintln(w, "Error occured")
				return
			}
			// Insert to owner node
			h.store.Put(key, req.Value)
			json.Unmarshal(bodyBytes, &req)

			// Replica 1

			rport1 := owner[1].PORT

			url := fmt.Sprintf(
				"http://localhost:%s/replica/%s",
				rport1,
				key,
			)

			replicaReq1, err := http.NewRequest(
				r.Method,
				url,
				bytes.NewReader(bodyBytes),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Copy Headers
			replicaReq1.Header = r.Header.Clone()
			// Send request
			repRes1, err := http.DefaultClient.Do(replicaReq1)
			if err != nil {
				http.Error(w, "Failed to replicate data", http.StatusBadGateway)
				return
			}
			// Close request
			defer repRes1.Body.Close()
			body, err := io.ReadAll(repRes1.Body)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Fprintln(w, string(body))

			// Replica 2r
			rport2 := owner[2].PORT

			url1 := fmt.Sprintf(
				"http://localhost:%s/replica/%s",
				rport2,
				key,
			)

			replicaReq2, err := http.NewRequest(
				r.Method,
				url1,
				bytes.NewReader(bodyBytes),
			)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Copy Headers
			replicaReq2.Header = r.Header.Clone()
			// Send request 2
			repRes2, err := http.DefaultClient.Do(replicaReq2)
			if err != nil {
				http.Error(w, "Failed to replicate data", http.StatusBadGateway)
				return
			}
			// Close request
			defer repRes2.Body.Close()
			body1, err := io.ReadAll(repRes2.Body)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Fprintln(w, string(body1))
			fmt.Fprintln(w, "Inserted data")
		case "DELETE":
			h.store.Delete(key)
			fmt.Fprintln(w, "Deleted Key")
		}

	} else {
		port := owner[0].PORT

		url := fmt.Sprintf(
			"http://localhost:%s/key/%s",
			port,
			key,
		)

		req, err := http.NewRequest(
			r.Method,
			url,
			r.Body,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req.Header = r.Header.Clone()

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Failed to forward request", http.StatusBadGateway)
			return
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Fprintln(w, string(body))
	}
}

func (h *Handler) HandleReplica(w http.ResponseWriter, r *http.Request) {

	key := strings.TrimPrefix(
		r.URL.Path,
		"/replica/",
	)

	var req model.PutRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		fmt.Fprintln(w, "Error occured")
		return
	}

	h.store.Put(key, req.Value)
	fmt.Fprintln(w, "Inserted data")

}

func (h *Handler) forwardRequest(owner cluster.Node, key string, w http.ResponseWriter, r *http.Request) {
	port := owner.PORT

	url := fmt.Sprintf(
		"http://localhost:%s/key/%s",
		port,
		key,
	)

	req, err := http.NewRequest(
		r.Method,
		url,
		r.Body,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header = r.Header.Clone()

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprintln(w, string(body))
}
