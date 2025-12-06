package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/hashicorp/mdns"
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

// TODO: how are these used?
const (
	serviceName = "provisioner-otp-service"
	domain      = ""
)

func (c *Cluster) init(outputCh chan string, ctx context.Context) {
	// Create Serf configuration
	config := serf.DefaultConfig()
	config.NodeName = c.Config.NodeName                  // TODO: set to agent ID
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
	log.Printf("Device Type: %s\n", c.Config.NodeType)
	if c.Config.NodeType == "worker" {
		log.Println("Joining an existing cluster...")
		// Discover BootBox address via mDNS
		serfAddress := receive(ctx)
		joinAddr := fmt.Sprintf("%s:%d", serfAddress, c.Config.BindPort)
		_, err := serfAgent.Join([]string{joinAddr}, true)
		if err != nil {
			log.Printf("Failed to join cluster at %s: %v", joinAddr, err)
		}
	} else {
		// Launch BootBox mDNS broadcaster
		go broadcast(ctx)
		fmt.Println("Running as BootBox...waiting for workers to join.") // TODO: debug message
	}

	// Wait a moment for the cluster to stabilize
	time.Sleep(2 * time.Second)

	// If device is worker, run requester to get token, else run responder indefinitely
	if c.Config.NodeType == "worker" {
		outputCh <- requester(serfAgent)
		serfAgent.Leave()
		serfAgent.Shutdown()
	} else {
		responder(eventCh, ctx) // blocks indefinitely
	}
}

// responder() listens for incoming queries and responds with a one-time token
func responder(eventCh chan serf.Event, ctx context.Context) {
	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			log.Println("Stopping responder on received signal")
			return
		case e := <-eventCh:
			if e.EventType() == serf.EventQuery {
				query := e.(*serf.Query)
				// We only respond to the specific query name
				if query.Name == "provisioner-otp" {
					log.Printf("Received query from %s", query.Name)

					// TODO: get token from API server
					token := "ONE-TIME-TOKEN-12345"
					query.Respond([]byte(token))
				}
			}
		}
	}
}

// requester() sends a query to the cluster requesting a one-time token
func requester(agent *serf.Serf) string {
	// Create query for OTP
	queryName := "provisioner-otp"
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
		log.Printf("Query failed: %v", err)
		return ""
	}

	// Process Responses
	for response := range resp.ResponseCh() {
		token := string(response.Payload)
		log.Printf("Token: %s From Node: %s\n", token, response.From)
		return token
	}
	return ""
}

// broadcast() starts an mDNS service to advertise a message.
func broadcast(ctx context.Context) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Failed to get hostname: %v", err)
		hostname = "unknown-host"
	}

	message := "Provisioner_Bootbox_OTP"

	// Setup service info with TXT records containing our message
	service, err := mdns.NewMDNSService(
		hostname,
		serviceName,
		domain,
		"",
		8080,
		getPhysIPs(),
		[]string{message},
	)
	if err != nil {
		log.Printf("Failed to create mDNS service: %v\n", err)
	}

	// Create the mDNS server
	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		log.Printf("Failed to create mDNS server: %v\n", err)
	}
	defer server.Shutdown()

	// Keep broadcasting until context is cancelled
	<-ctx.Done()
	log.Println("Stopping broadcast on received signal")
}

// receive() searches for mDNS services and returns the IP
// address of the first matching service.
func receive(ctx context.Context) string {
	log.Printf("Searching for mDNS services: %s", serviceName)

	// Create a channel to receive service entries
	entriesCh := make(chan *mdns.ServiceEntry, 10)
	complete := make(chan string, 1)

	// Goroutine to process entries
	go func() {
		for entry := range entriesCh {
			log.Printf("Discovered service: %s\n", entry.Name)
			if entry.Info == "Provisioner_Bootbox_OTP" {
				complete <- entry.AddrV4.String()
			}
		}
	}()

	// Continuously search for services
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case ipAddress := <-complete:
			close(entriesCh)
			return ipAddress
		case <-ctx.Done():
			close(entriesCh)
			return ""
		case <-ticker.C:
			mdns.Lookup(serviceName, entriesCh)
		}
	}
}

// getPhysIPs() returns IP addresses for physical network interfaces,
// filtering out loopback, virtual, and down interfaces.
func getPhysIPs() []net.IP {
	interfaces, _ := net.Interfaces()
	var result []net.IP

	// Loop through all network interfaces
	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Filter out virtual interfaces (docker, virtual box, vmware, etc.)
		name := iface.Name
		if len(name) >= 6 && (name[:6] == "docker" || name[:3] == "vir" ||
			name[:4] == "veth" || name[:2] == "br" || name[:3] == "vmn") {
			continue
		}

		// Get addresses for this interface
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		// Build list of IPv4IPs
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// Only include IPv4 addresses
			if ip != nil && ip.To4() != nil {
				result = append(result, ip)
			}
		}
	}

	return result
}
