# step_2

This repository provides a simple container runtime written in Go. It leverages Linux namespaces, cgroups, and chroot to isolate processes. Below is an overview of how to use and understand the code.

## Overview

- **Namespaces:** The runtime uses namespaces (UTS, PID, Network, Mount, IPC) to isolate resources like hostname, processes, network, and mounts.
- **Cgroups:** It applies cgroups to limit process counts, memory usage, and CPU shares for isolated containers.
- **Chroot & Mounts:** The code sets up a new root filesystem and mounts essential directories such as `/proc` and `/tmp`.
- **OverlayFS:** The runtime now supports using OverlayFS for layered filesystems.
- **Bind Mounts:** The runtime supports bind mounting directories into the container.

## Building

Make sure you have Go installed. Then run:

```bash
cd /home/grass/projects/congo/step_2
go build -o container-runtime main.go
```

## Usage

Run the following command to create a new container environment:

```bash
go run main.go run <rootfs> <process_limit> <memory_limit> <cpu_share> <env_vars> [--mount source:dest[:ro]] [--layers layer1,layer2] -- <cmd> <args>
```
Where:
- `/path/to/rootfs` is the root filesystem for the container.
- `100` is the process limit.
- `512m` is the memory limit (in megabytes).
- `1024` is the CPU share.
- `KEY=VALUE,FOO=BAR` specifies environment variables.
- `[--mount source:dest[:ro]]` specifies bind mounts (optional).
- `[--layers layer1,layer2]` specifies OverlayFS layers (optional).
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

## new changes

1. **OverlayFS Support**: The runtime now supports using OverlayFS for layered filesystems.
2. **Bind Mounts**: The runtime supports bind mounting directories into the container.
3. **Configuration Parsing**: The `parseConfig` function has been updated to handle new command-line arguments for bind mounts and OverlayFS layers.
4. **Setup Functions**: New functions `setupLayeredRootfs` and `performBindMounts` have been added to handle the setup of OverlayFS and bind mounts, respectively.
5. **Cleanup Function**: A `cleanup` function has been added to clean up cgroups and unmount filesystems after the container exits.
6. **Updated Usage**: The usage instructions now include options for bind mounts and OverlayFS layers.

## Notes

- **Security:** This is a simplified demonstration and may not handle all security concerns of a production container runtime.
- **Privileges:** Running this code requires root privileges because it sets up namespaces and cgroups.

## Troubleshooting

- Check logs for messages prefixed with `container-runtime:` if something fails.
- Ensure your kernel supports required namespaces and cgroup controllers.