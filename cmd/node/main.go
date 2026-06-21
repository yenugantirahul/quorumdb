package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yenuganti/quorumdb/internal/api"
	"github.com/yenuganti/quorumdb/internal/cluster"
	"github.com/yenuganti/quorumdb/internal/hash"
	"github.com/yenuganti/quorumdb/internal/storage"
)

func main() {

	// port := os.Getenv("PORT")

	nodeID := os.Args[1]
	// Create storage engine
	store, err := storage.NewBadgerStore("./data" + nodeID)

	portStr := os.Args[2]
	self := cluster.Node{
		ID:      nodeID,
		Address: "localhost",
		PORT:    portStr,
	}

	ring := hash.NewHashRing()

	node1 := cluster.Node{
		ID:   "node1",
		PORT: "8080",
	}
	node2 := cluster.Node{
		ID:   "node2",
		PORT: "8081",
	}
	node3 := cluster.Node{
		ID:   "node3",
		PORT: "8082",
	}

	ring.AddNode(node1)
	ring.AddNode(node2)
	ring.AddNode(node3)

	if err != nil {
		log.Fatal(err)
	}

	// Create handler and inject storage
	handler := api.NewHandler(store, ring, self)

	// Register routes
	http.HandleFunc("/health/", handler.Health)
	http.HandleFunc("/key/", handler.HandleKey)
	http.HandleFunc("/replica/", handler.HandleReplica)

	log.Println("Server running on :" + self.PORT)
	hashvalue := ring.GetNode("user1")
	log.Println(hashvalue)

	// Start server
	err = http.ListenAndServe(":"+self.PORT, nil)

	if err != nil {
		log.Fatal(err)
	}
}
