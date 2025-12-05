package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/serf/serf"
)

type ClusterConfig struct {
	BindPort   int
	BindAddr   string
	NodeName   string
	NodeType   string
	DeviceType string
}

type Cluster struct {
	Config    ClusterConfig
	SerfAgent *serf.Serf
	EventCh   chan serf.Event
}

func (c *Cluster) init(outputCh chan string) {

	// TODO: this is more of a note. To get to run on different ,addrs, you need to set the
	// BindAddr to the specific IP of the interface you want to use, not 127.0.0.1. Same on
	// other nodes. For testing locally, you can use different ports on localhost.

	// Create Serf configuration
	config := serf.DefaultConfig()
	config.NodeName = c.Config.NodeName
	config.MemberlistConfig.BindAddr = c.Config.BindAddr // Change to your machine's IP address or 127.0.0.1 for localhost
	config.MemberlistConfig.BindPort = c.Config.BindPort // You use the same port when on different machines

	// Channel stuff
	eventCh := make(chan serf.Event, 256)
	config.EventCh = eventCh

	// Create a new Serf instance
	serfAgent, err := serf.Create(config)
	if err != nil {
		log.Fatalf("Failed to create Serf agent: %v", err)
	}
	defer serfAgent.Leave()
	defer serfAgent.Shutdown()

	// Join existing cluster if Worker node
	//joinArgs := flag.Args()
	log.Printf("Device Type: %s\n", c.Config.NodeType)
	if c.Config.NodeType == "worker" {
		log.Println("Joining an existing cluster...")
		joinAddr := "192.168.1.35:7946" //joinArgs[0] TODO: replace w/ broadcast address
		_, err := serfAgent.Join([]string{joinAddr}, true)
		if err != nil {
			log.Printf("Failed to join cluster at %s: %v", joinAddr, err)
		} else {
			fmt.Printf("Successfully joined cluster at %s\n", joinAddr)
		}
	} else {
		fmt.Println("Running as BootBox...waiting for workers to join.") // TODO: debug message
	}

	// Wait a moment for the cluster to stabilize
	time.Sleep(2 * time.Second)

	// If device is worker, run requester to get token, else run responder indefinitely
	if c.Config.NodeType == "worker" {
		outputCh <- requester(serfAgent)
		serfAgent.Leave()
		serfAgent.Shutdown()
		log.Println("Worker node finished! Exiting init.")
	} else {
		responder(eventCh) // blocks indefinitely
		log.Println("BootBox node finished! Exiting init.")
	}
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

					// TODO: get token from API server
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

func requester(agent *serf.Serf) string {
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
			return token
		}
	}
	return ""
}
