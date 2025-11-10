# ConGo - Container Runtime in Golang

lightweight, educational container runtime written in Go, designed to demonstrate the core concepts behind containerization on Linux. It leverages Linux namespaces and cgroups to create isolated environments for running applications.

## Features

- **Container Lifecycle Management:** Create, start, stop, restart, and remove containers.
- **Process Isolation:** Uses Linux namespaces (PID, UTS, NS, NET, IPC, USER) to isolate container processes.
- **Resource Management:** Utilizes cgroups to limit container resources like CPU, memory, and PIDs.
- **Filesystem Isolation:** Mounts a root filesystem for each container.
- **Container Images:** Basic support for creating images from containers (`commit`).
- **Networking:** Basic network setup for containers.
- **Volume Mounting:** Supports mounting host directories into containers.
- **Interactive Shell:** Get an interactive shell inside a running container.
- **Logging:** View logs from containers.

## Project Goal

The primary goal of ConGo is to serve as a learning tool for understanding how containers work under the hood. It is not intended for production use but rather as a way to explore the technologies that make containers possible.

## Getting Started

### Prerequisites

- Go (version 1.21 or later)
- Linux Kernel with support for namespaces and cgroups

### Building and Running with Make

To simplify building and running, you can use the provided `Makefile`.

**Build the binary:**
```sh
make build
```

**Run a container:**
The `make run` command automatically downloads the Alpine rootfs and provides the path to the container. Pass your desired command and arguments using the `ARGS` variable.

```sh
# Run a command and print "Hello from container"
make run ARGS="--hostname my-alpine /bin/echo Hello from container"

# Start an interactive shell
make run ARGS="-i --hostname my-alpine /bin/sh"
```

**List containers:**
```sh
make ps
```

For more commands, see the [Usage Guide](./usage.md) and the `Makefile`.

## Usage

For detailed usage instructions, please see [usage.md](./usage.md).

## Internals

For an overview of the internal architecture and components of ConGo, please see [internals/internals.md](./internals/internals.md).

## Disclaimer

ConGo is an experimental project. It is not secure and should not be used for any production workloads.

