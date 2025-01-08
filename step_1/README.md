# Container Runtime README

This repository provides a simple container runtime written in Go. It leverages Linux namespaces, cgroups, and chroot to isolate processes. Below is an overview of how to use and understand the code.

## Overview

- **Namespaces:** The runtime uses namespaces (UTS, PID, Network, Mount, IPC) to isolate resources like hostname, processes, network, and mounts.  
- **Cgroups:** It applies cgroups to limit process counts, memory usage, and CPU shares for isolated containers.  
- **Chroot & Mounts:** The code sets up a new root filesystem and mounts essential directories such as `/proc` and `/tmp`.

## Structure

- **main.go:**  
  - `run()` spawns a new process with the required flags to enter namespaces.  
  - `child()` sets up the container environment and executes the user command.  
  - `parseConfig()` parses command-line arguments, including resource limits and environment variables.  
  - `setupContainer()` configures environment variables, hostname, and cgroups.  
  - `setupRootfs()` mounts and prepares the root filesystem structure.  
  - `setupCgroups()` enforces resource limits using cgroups.  
  - `validateConfig()` checks valid paths, format of numeric arguments, etc.

## Building

Make sure you have Go installed. Then run:

```bash
cd /home/grass/projects/rust/congo/step_1
go build -o container-runtime main.go
```

## Usage

Run the following command to create a new container environment:

```bashs
go run main.go run <rootfs> <process_limit> <memory_limit> <cpu_share> <env_vars> -- <cmd> <args>

```
Where:
- `/path/to/rootfs` is the root filesystem for the container.
- `100` is the process limit.
- `512m` is the memory limit (in megabytes).
- `1024` is the CPU share.
- `KEY=VALUE,FOO=BAR` specifies environment variables.
- `/bin/bash` (or similar) is the program to execute inside the container.

## Root Filesystem Setup

In order for the chroot setup to work, you need to create the directory and populate it with the rootfs. For a Debian-based rootfs:

```bash
sudo apt-get install debootstrap
sudo debootstrap stable /home/pj/ubuntufs http://deb.debian.org/debian/
```

Or using Alpine Linux:

```bash
wget https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-latest-x86_64.tar.gz
sudo tar -xzf alpine-minirootfs-latest-x86_64.tar.gz -C /home/liz/ubuntufs
```

## Notes

- **Security:** This is a simplified demonstration and may not handle all security concerns of a production container runtime.  
- **Privileges:** Running this code requires root privileges because it sets up namespaces and cgroups.

## Troubleshooting

- Check logs for messages prefixed with `container-runtime:` if something fails.  
- Ensure your kernel supports required namespaces and cgroup controllers.