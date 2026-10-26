package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yenuganti/quorumdb/internal/api"
	"github.com/yenuganti/quorumdb/internal/cluster"
	"github.com/yenuganti/quorumdb/internal/hash"
	"github.com/yenuganti/quorumdb/internal/storage"
	"github.com/yenuganti/quorumdb/internal/version"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run ./cmd/server <nodeID> <port>")
	}

	nodeID := os.Args[1]
	port := os.Args[2]

	// This represents the current process.
	self := cluster.Node{
		ID:    nodeID,
		PORT:  port,
		Alive: true,
	}

	// Every process knows about the complete cluster.
	nodes := []cluster.Node{
		{
			ID:    "node1",
			PORT:  "8080",
			Alive: true,
		},
		{
			ID:    "node2",
			PORT:  "8081",
			Alive: true,
		},
		{
			ID:    "node3",
			PORT:  "8082",
			Alive: true,
		},
	}

	// Each node gets its own BadgerDB directory.
	store, err := storage.NewBadgerStore("./data/" + nodeID)
	if err != nil {
		log.Fatal(err)
	}

	manager := cluster.NewManager()
	ring := hash.NewHashRing()
	versionManager := version.NewManager()

	// Add every cluster node to the manager and hash ring.
	for _, node := range nodes {
		manager.AddNode(node)
		ring.AddNode(node)
	}

	handler := api.NewHandler(
		store,
		ring,
		self,
		manager,
		versionManager,
	)

	healthManager := cluster.NewHealthManager(
		manager,
		self,
	)

	healthManager.Start()

	http.HandleFunc("/health", handler.HandleHealth)
	http.HandleFunc("/key/", handler.HandleKey)
	http.HandleFunc("/replica/", handler.HandleReplica)

	log.Println("Current node:", self.ID)
	log.Println("Current port:", self.PORT)

	log.Println("Known cluster nodes:")
	for _, node := range nodes {
		log.Printf("- %s running on port %s\n", node.ID, node.PORT)
	}

	err = http.ListenAndServe(":"+self.PORT, nil)
	if err != nil {
		log.Fatal(err)
	}
}
