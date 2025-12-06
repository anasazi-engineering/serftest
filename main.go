package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Command line flags
	var c Cluster
	bindPort := flag.Int("p", 7946, "Port to bind the Serf agent to")
	nodeName := flag.String("n", "bootbox001", "Name of the Serf agent")
	nodeType := flag.String("t", "bootbox", "Type of the Serf agent (bootbox or worker)")
	flag.Parse()

	// Configure cluster
	c.Config = ClusterConfig{
		BindPort: *bindPort, // TODO: change to configurable port
		NodeName: *nodeName, // TODO: Use AgentID
		NodeType: *nodeType, // TODO: from config
	}

	// Start and/or join cluster
	outputCh := make(chan ProvConfig, 1)
	c.init(outputCh, ctx)
	if c.Config.NodeType == "worker" {
		fmt.Println("Worker node running...waiting for token response")
		payload := <-outputCh // Worker blocks until it receives the token
		fmt.Printf("Worker node received URL: %s, Token: %s\n", payload.BaseURL, payload.OTP)
	}

	// The End
	fmt.Println("\nEnd of main(). Press Ctrl+C to stop the agent")
	// sigCh := make(chan os.Signal, 1)
	// signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
}
