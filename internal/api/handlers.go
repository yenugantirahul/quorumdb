package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/yenuganti/quorumdb/internal/cluster"
	"github.com/yenuganti/quorumdb/internal/hash"
	"github.com/yenuganti/quorumdb/internal/model"
	"github.com/yenuganti/quorumdb/internal/storage"
	"github.com/yenuganti/quorumdb/internal/version"
)

// Handler holds the store, consistent hash ring, and this node's identity.
type Handler struct {
	store   storage.Store
	ring    *hash.HashRing
	self    cluster.Node
	manager *cluster.Manager
	version *version.Manager
}

const (
	ReadQuorum  = 2
	WriteQuorum = 2
)

func NewHandler(
	store storage.Store,
	ring *hash.HashRing,
	self cluster.Node,
	manager *cluster.Manager,
	version *version.Manager,

) *Handler {
	return &Handler{
		store:   store,
		ring:    ring,
		self:    self,
		manager: manager,
		version: version,
	}
}

// HandleKey routes GET / PUT / DELETE requests to the correct node or handles
// them locally if this node is the primary owner of the key.
func (h *Handler) HandleKey(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/key/")
	if key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	owner := h.ring.GetNode(key)

	// Forward first, then handle locally — avoids the duplicate
	// inline forwarding block that existed in the original code.

	switch r.Method {
	// GET Method
	case http.MethodGet:

		h.handleGet(w, key, owner)
		// PUT Method
	case http.MethodPut:
		if owner[0].ID != h.self.ID {
			h.forwardRequest(owner[0], key, w, r)
			return
		}

		h.handlePut(w, r, key, owner)

	case http.MethodDelete:
		if owner[0].ID != h.self.ID {
			h.forwardRequest(owner[0], key, w, r)
			return
		}
		h.handleDelete(w, key, owner)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// Gets the value for the key
func (h *Handler) handleGet(
	w http.ResponseWriter,
	key string,
	owner []cluster.Node,
) {

	readCount := 0
	var value model.Record

	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, node := range owner {
		if node.ID != h.self.ID && !h.manager.IsAlive(node.ID) {
			continue
		}
		wg.Add(1)
		go func(node cluster.Node) {
			defer wg.Done()
			var (
				record model.Record
				err    error
			)

			if node.ID == h.self.ID {
				record, err = h.store.Get(key)
			} else {
				record, err = h.readFromReplica(node, key)
			}
			mu.Lock()
			if err == nil {
				readCount++

				// Keep the first successful value
				if record.Version > value.Version {
					value = record
				}
			}
			mu.Unlock()
		}(node)
	}
	wg.Wait()
	// Read quorum achieved

	if readCount < 2 {
		http.Error(
			w,
			"read quorum not achieved",
			http.StatusServiceUnavailable,
		)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, value.Value)
}

// Reads From Replicas
func (h *Handler) readFromReplica(
	node cluster.Node,
	key string,
) (model.Record, error) {

	url := fmt.Sprintf(
		"http://localhost:%s/replica/%s",
		node.PORT,
		key,
	)

	resp, err := http.Get(url)
	if err != nil {
		return model.Record{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.Record{}, fmt.Errorf("read failed")
	}

	var record model.Record

	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return model.Record{}, err
	}

	return record, nil
}

// This function handles PUT request
func (h *Handler) handlePut(w http.ResponseWriter, r *http.Request, key string, owner []cluster.Node) {
	// Read body once so we can  decode and forward it to replicas.

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req model.PutRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// json.Unmarshal silently left req.Value as "".
	if req.Value == "" {
		http.Error(w, `"value" field is required and must not be empty`, http.StatusBadRequest)
		return
	}
	record := model.Record{
		Value:   req.Value,
		Version: h.version.Next(), // Temporary
	}

	if err := h.store.Put(key, record); err != nil {
		http.Error(w, "failed to store value", http.StatusInternalServerError)
		return
	}

	// Replicate to every replica node (owner[1:]).
	ack := 1 // Primary already acknowledged the write.

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, replica := range owner[1:] {

		if !h.manager.IsAlive(replica.ID) {
			continue
		}

		wg.Add(1)

		go func(replica cluster.Node) {
			defer wg.Done()

			if err := h.replicateTo(replica, key, bodyBytes); err != nil {
				fmt.Printf("replication failed to %s: %v\n", replica.ID, err)
				return
			}

			mu.Lock()
			ack++
			mu.Unlock()

			fmt.Printf("replicated to %s\n", replica.ID)

		}(replica)
	}

	wg.Wait()

	if ack < WriteQuorum {
		http.Error(w, "write quorum not achieved", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "ok")

}

// Deletes from replicas

func (h *Handler) replicateDelete(node cluster.Node, key string) error {
	url := fmt.Sprintf(
		"http://localhost:%s/replica/%s",
		node.PORT,
		key,
	)

	req, err := http.NewRequest(
		http.MethodDelete,
		url,
		nil,
	)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("replica returned %d", res.StatusCode)
	}

	return nil
}

func (h *Handler) handleDelete(w http.ResponseWriter,
	key string,
	owner []cluster.Node) {
	if err := h.store.Delete(key); err != nil {
		http.Error(w, fmt.Sprintf("key not found: %s", key), http.StatusNotFound)
		return
	}
	ack := 1 // Primary already acknowledged the write.

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, replica := range owner[1:] {

		if !h.manager.IsAlive(replica.ID) {
			continue
		}

		wg.Add(1)

		go func(replica cluster.Node) {
			defer wg.Done()

			if err := h.replicateDelete(replica, key); err != nil {
				fmt.Printf("Deletion failed to %s: %v\n", replica.ID, err)
				return
			}

			mu.Lock()
			ack++
			mu.Unlock()

			fmt.Printf("Deleted from  %s\n", replica.ID)

		}(replica)
	}

	wg.Wait()

	if ack < WriteQuorum {
		http.Error(w, "write quorum not achieved", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Deleted")

}

// returns a real error instead of just printing.
func (h *Handler) replicateTo(replica cluster.Node, key string, body []byte) error {
	url := fmt.Sprintf("http://localhost:%s/replica/%s", replica.PORT, key)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build replica request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send replica request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("replica %s returned status %d", replica.ID, res.StatusCode)
	}
	return nil
}

// HandleReplica accepts PUT requests from the primary and writes them locally.
// It does NOT re-replicate — only the primary drives replication.
func (h *Handler) HandleReplica(w http.ResponseWriter, r *http.Request) {

	key := strings.TrimPrefix(r.URL.Path, "/replica/")
	if key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:

		var req model.PutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.Value == "" {
			http.Error(w, `"value" field is required and must not be empty`, http.StatusBadRequest)
			return
		}

		record := model.Record{
			Value:   req.Value,
			Version: h.version.Next(), // Temporary
		}

		if err := h.store.Put(key, record); err != nil {
			http.Error(w, "failed to store value", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	case http.MethodGet:
		record, err := h.store.Get(key)
		if err != nil {
			http.Error(w, "failed to get value", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(record); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	case http.MethodDelete:

		if err := h.store.Delete(key); err != nil {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "deleted")
	}
}

// status code and (b) appended an extra newline. io.Copy is correct here.
func (h *Handler) forwardRequest(owner cluster.Node, key string, w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("http://localhost:%s/key/%s", owner.PORT, key)

	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header = r.Header.Clone()

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "failed to forward request", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	// Propagate headers and status from the upstream node.
	for k, vals := range res.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body) //nolint:errcheck
}
