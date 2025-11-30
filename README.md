# Serf Agent

A simple Go application that creates a single agent and joins a Serf cluster.

## Setup

Install dependencies:
```bash
go mod tidy
```

## Usage

	// TODO: this is more of a note. To get to run on different ,addrs, you need to set the
	// BindAddr to the specific IP of the interface you want to use, not 127.0.0.1. Same on
	// other nodes. For testing locally, you can use different ports on localhost.

### Run locally as a BootBox:

```bash
$ ./serf1
```

### Run locally as a Worker and join BootBox cluster

For example, to join a cluster on the same machine, with an agent at `127.0.0.1:7946`, you must use a different port number than the BootBox.
```bash
$ ./serf1 -t worker -p 7947 -n worker001 127.0.0.1:7946
```

### Run BootBox on external interface

If running BootBox and Workers on separate devices, then agent must bind to external network interfaces, not localhost.

```bash
$ ./serf1 -a 192.168.1.35
```

### Connect Worker to BootBox on separate device

If running BootBox and Workers on separate devices, then agent must bind to external network interfaces, not localhost.

```bash
$ ./serf1 -a 192.168.1.22 -t worker -n worker001 192.168.1.35:7946
```
