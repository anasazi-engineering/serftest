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
$ ./serf1
```


For example, to join a cluster on the same machine, with an agent at `127.0.0.1:7946`, node name of 'mike', and using port '7947':
```bash
$ ./serf1 -p 7947 -n mike 127.0.0.1:7946
```

## Testing LOCALLY with Multiple Agents

To test clustering locally, you must change the port in line 18 and recompile. Then open terminal for each custom binary with different hardcoded port. I assuming that agents on different nodes can connect with the same port?

## Features

- Creates a Serf agent with configurable node name
- Joins existing clusters when provided an address
- Displays current cluster members
- Graceful shutdown on Ctrl+C
