package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/serf/serf"
)

func main() {
	// Parse command line flags
	bindPort := flag.Int("p", 7946, "Port to bind the Serf agent to")
	nodeName := flag.String("n", "agent007", "Name of the Serf agent")
	flag.Parse()

	// Create Serf configuration
	config := serf.DefaultConfig()
	config.NodeName = *nodeName
	config.MemberlistConfig.BindAddr = "127.0.0.1"
	config.MemberlistConfig.BindPort = *bindPort

	// Create a new Serf instance
	serfAgent, err := serf.Create(config)
	if err != nil {
		log.Fatalf("Failed to create Serf agent: %v", err)
	}
	defer serfAgent.Shutdown()

	fmt.Printf("Serf agent '%s' started successfully\n", config.NodeName)
	fmt.Printf("Listening on %s:%d\n", config.MemberlistConfig.BindAddr, config.MemberlistConfig.BindPort)

	// Join existing cluster if specified
	joinArgs := flag.Args()
	log.Println("Length of 'flag.Args()': ", len(joinArgs))
	if len(joinArgs) > 0 {
		joinAddr := joinArgs[0]
		_, err := serfAgent.Join([]string{joinAddr}, true)
		if err != nil {
			log.Printf("Failed to join cluster at %s: %v", joinAddr, err)
		} else {
			fmt.Printf("Successfully joined cluster at %s\n", joinAddr)
		}
	} else {
		fmt.Println("Running as standalone agent. Pass an address to join a cluster.")
	}

	// Display current members
	members := serfAgent.Members()
	fmt.Printf("\nCurrent cluster members (%d):\n", len(members))
	for _, member := range members {
		fmt.Printf("  - %s (%s)\n", member.Name, member.Addr)
	}

	// Wait for interrupt signal to gracefully shutdown
	fmt.Println("\nPress Ctrl+C to stop the agent")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
}
