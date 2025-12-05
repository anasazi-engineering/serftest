package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Command line flags
	var c Cluster
	c.Config = ClusterConfig{
		BindPort: *flag.Int("p", 7946, "Port to bind the Serf agent to"),                     // TODO: change to configurable port
		BindAddr: *flag.String("a", "127.0.0.1", "Address to bind the Serf agent to"),        // TODO: discover address
		NodeName: *flag.String("n", "bootbox001", "Name of the Serf agent"),                  // TODO: Use AgentID
		NodeType: *flag.String("t", "bootbox", "Type of the Serf agent (bootbox or worker)"), // TODO: from config
	}
	flag.Parse()
	fmt.Printf("Starting Serf agent with name: %s, type: %s, address: %s:%d\n",
		c.Config.NodeName, c.Config.NodeType, c.Config.BindAddr, c.Config.BindPort)

	// Start and/or join cluster
	outputCh := make(chan string, 1)
	c.init(outputCh)
	if c.Config.NodeType == "worker" {
		fmt.Println("Worker node running...waiting for token response")
		token := <-outputCh // block here on channel
		fmt.Printf("Worker node received token: %s\n", token)
	}

	fmt.Println("\nEnd of main(). Press Ctrl+C to stop the agent")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
