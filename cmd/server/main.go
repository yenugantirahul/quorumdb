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
		log.Fatal("Usage: go run main.go <nodeID> <port>")
	}

	nodeID := os.Args[1]
	port := os.Args[2]

	self := cluster.Node{
		ID:    nodeID,
		PORT:  port,
		Alive: true,
	}

	store, err := storage.NewBadgerStore("./data/" + nodeID)
	if err != nil {
		log.Fatal(err)
	}

	manager := cluster.NewManager()
	versionManager := version.NewManager()

	// Add nodes to the manager

	manager.AddNode(self)

	ring := hash.NewHashRing()

	// Add nodes to hashring

	ring.AddNode(self)

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

	// Start the health checker

	healthManager.Start()

	// Routes
	http.HandleFunc("/health", handler.HandleHealth)
	http.HandleFunc("/key/", handler.HandleKey)
	http.HandleFunc("/replica/", handler.HandleReplica)

	log.Println("Node:", self.ID)
	log.Println("Port:", self.PORT)

	// ----------------------------
	// Start Server
	// ----------------------------
	err = http.ListenAndServe(":"+self.PORT, nil)
	if err != nil {
		log.Fatal(err)
	}
}
