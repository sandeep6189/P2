// DO NOT MODIFY!

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/cmu440-F17/paxosapp/tests_final/proxy"
)

var (
	paxosPorts = flag.Int("paxosport", 9000, "real ports of this node")
	proxyPort  = flag.Int("proxyport", 9001, "proxy ports of this node")
	nodeID     = flag.Int("id", 0, "node ID must match index of this node's port in the ports list")
	numRetries = flag.Int("retries", 5, "number of times a node should retry dialing another node")
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)
}

func main() {
	flag.Parse()

	// Set up the proxy
	for i := 0; i <= *numRetries; i++ {
		if i == *numRetries {
			fmt.Errorf("could not instantiate proxy on port %d", *proxyPort)
		}
		_, err := proxy.NewProxy(*paxosPorts, *proxyPort, *nodeID)
		if err != nil {
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	// Run the paxos node forever
	select {}
}
