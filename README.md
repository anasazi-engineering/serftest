# Serf Agent

A simple Go application that creates a single agent and joins a Serf cluster.

## Setup

Install dependencies:
```bash
go mod tidy
```

## Usage

### Run as standalone agent:
```bash
go run main.go
```

### Join an existing cluster:
```bash
go run main.go <address:port>
```

For example, to join a cluster with an agent at `127.0.0.1:7946`, node name of 'mike', and using port '7947':
```bash
$ ./serf1 -p 7947 -n mike 127.0.0.1:7946
```

## Testing with Multiple Agents

To test clustering locally, you must change the port in line 18 and recompile. Then open terminal for each custom binary with different hardcoded port.

**Terminal 1** (first agent):
```bash
go run main.go
```

**Terminal 2** (second agent, joins first):
```bash
# Modify the port in code or use environment variables
go run main.go 127.0.0.1:7946
```

## Features

- Creates a Serf agent with configurable node name
- Joins existing clusters when provided an address
- Displays current cluster members
- Graceful shutdown on Ctrl+C
