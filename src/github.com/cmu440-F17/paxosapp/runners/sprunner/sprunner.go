// DO NOT MODIFY!

package main

import (
	"flag"
	"log"
	"strings"

	"github.com/cmu440-F17/paxosapp/staffnode"
)

var (
	ports      = flag.String("ports", "", "ports for all paxos nodes")
	numNodes   = flag.Int("N", 1, "the number of nodes in the ring")
	nodeID     = flag.Int("id", 0, "node ID must match index of this node's port in the ports list")
	numRetries = flag.Int("retries", 10, "number of times a node should retry dialing another node")
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
}

func main() {
	flag.Parse()

	portStrings := strings.Split(*ports, ",")

	hostMap := make(map[int]string)
	for i, port := range portStrings {
		hostMap[i] = "localhost:" + port
	}

	// Create and start the Staff Paxos Node.
	_, err := staffnode.NewStaffNode(hostMap[*nodeID], hostMap, *numNodes, *nodeID, *numRetries, false)
	if err != nil {
		log.Fatalln("Failed to create staff paxos node:", err)
	}

	// Run the paxos node forever.
	select {}
}
