package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/serf/serf"
)

func main() {
	// Parse command line flags
	bindPort := flag.Int("p", 7946, "Port to bind the Serf agent to")
	nodeName := flag.String("n", "agent007", "Name of the Serf agent")
	flag.Parse()

	// TODO: this is more of a note. To get to run on different ,addrs, you need to set the
	// BindAddr to the specific IP of the interface you want to use, not 127.0.0.1. Same on
	// other nodes. For testing locally, you can use different ports on localhost.

	// Create Serf configuration
	config := serf.DefaultConfig()
	config.NodeName = *nodeName
	config.MemberlistConfig.BindAddr = "192.168.1.35" // Change to your machine's IP address or 127.0.0.1 for localhost
	config.MemberlistConfig.BindPort = *bindPort      // You use the same port when on different machines
	isResponder := config.NodeName == "agent007"

	// Channel stuff
	eventCh := make(chan serf.Event, 256)
	config.EventCh = eventCh

	if isResponder {
		log.Println("This node is set as responder")
	} else {
		log.Println("This node is a requester")
	}

	// Create a new Serf instance
	serfAgent, err := serf.Create(config)
	if err != nil {
		log.Fatalf("Failed to create Serf agent: %v", err)
	}
	defer serfAgent.Leave()
	defer serfAgent.Shutdown()

	fmt.Printf("Serf agent '%s' started successfully\n", config.NodeName)
	fmt.Printf("Listening on %s:%d\n", config.MemberlistConfig.BindAddr, config.MemberlistConfig.BindPort)

	// Join existing cluster if specified
	joinArgs := flag.Args()
	log.Println("Length of 'flag.Args()': ", len(joinArgs))
	if len(joinArgs) > 0 {
		log.Println("Joining an existing cluster...")
		joinAddr := joinArgs[0]
		_, err := serfAgent.Join([]string{joinAddr}, true)
		if err != nil {
			log.Printf("Failed to join cluster at %s: %v", joinAddr, err)
		} else {
			fmt.Printf("Successfully joined cluster at %s\n", joinAddr)
		}
	} else {
		fmt.Println("Running as standalone agent...waiting for others to join.")
	}

	// Wait a moment for the cluster to stabilize
	time.Sleep(2 * time.Second)

	// Display current members
	members := serfAgent.Members()
	fmt.Printf("\nCurrent cluster members (%d):\n", len(members))
	for _, member := range members {
		fmt.Printf("  - %s (%s)\n", member.Name, member.Addr)
	}

	// Start responder or requester based on node name
	if isResponder {
		go responder(eventCh)
	} else {
		go requester(serfAgent)
	}

	// Wait for interrupt signal to gracefully shutdown
	fmt.Println("\nPress Ctrl+C to stop the agent")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")
}

func responder(eventCh chan serf.Event) {
	for {
		select {
		case e := <-eventCh:
			if e.EventType() == serf.EventQuery {
				query := e.(*serf.Query)

				// We only respond to the specific query name
				if query.Name == "provisioner-OTP" {
					log.Printf("Received query from %s", query.Name)

					// Generate a one-time token (for demonstration, using a static token)
					token := "ONE-TIME-TOKEN-12345"

					// Respond to the query
					//err := serfAgent.Respond(query.ID, []byte(token))
					err := query.Respond([]byte(token))
					if err != nil {
						log.Printf("Failed to respond to query: %v", err)
					} else {
						log.Printf("Responded to query '%s' with token", query.Name)
					}
				}
			}
		}
	}
}

func requester(agent *serf.Serf) {
	// Create query for OTP
	queryName := "provisioner-OTP"
	queryPayload := []byte("Gimme OTP!")

	// Send the query to the entire cluster
	resp, err := agent.Query(
		queryName,
		queryPayload,
		&serf.QueryParam{
			FilterNodes: []string{}, // Target all members
			Timeout:     5 * time.Second,
		},
	)

	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}

	// --- Process Responses ---
	fmt.Println("\n## Processing Query Responses:")

	for response := range resp.ResponseCh() {
		token := string(response.Payload)

		fmt.Printf("Response From Node: **%s**\n", response.From)

		if token == "DENIED: Token already issued" {
			fmt.Printf("   Status: **Denied!** Token was already claimed by another requester.\n")
		} else {
			fmt.Printf("   Status: **Success!** Received Token: **%s**\n", token)
		}
	}

}
