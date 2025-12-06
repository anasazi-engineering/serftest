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


### Run BootBox on external interface

Network interface and IP address is automatically detected.

```bash
$ ./serf1 -t bootbox -n bootbox007
```

### Connect Worker to BootBox on separate device

Network interface and IP address is automatically detected.

```bash
$ ./serf1 -t worker -n worker001

```
