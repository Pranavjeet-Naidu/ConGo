# Usage Instructions

## Prerequisites

1. Ensure you have Go installed on your system.
2. Create a root filesystem directory and populate it with the root filesystem.
3. Make sure you have the necessary permissions to create network bridges and modify iptables rules.

### Creating a Root Filesystem

#### Using debootstrap:
```sh
sudo apt-get install debootstrap 
sudo debootstrap stable /path/to/rootfs http://deb.debian.org/debian/
```

#### Using Alpine Linux:
```sh
wget https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-latest-x86_64.tar.gz
sudo mkdir -p /path/to/rootfs
sudo tar -xzf alpine-minirootfs-latest-x86_64.tar.gz -C /path/to/rootfs
```

## Building

Compile the container runtime:

```bash
cd /home/grass/projects/congo/step_4
go build -o container main.go
```

## Running the Container

The basic command structure is:

```bash
sudo ./container PATH HOME USER SHELL TERM LANG [options] -- <command> [args]
```

Where the environment variables are:
- `PATH`: Path variable for the container
- `HOME`: Home directory in the container
- `USER`: Username in the container
- `SHELL`: Shell to use
- `TERM`: Terminal type
- `LANG`: Language setting

### Options

- `--mount <source> <destination> <ro|rw>`: Mount a host directory into the container
- `--user <username>`: Run the container as a specific user
- `--cap-add <capability>`: Add a Linux capability (e.g., CAP_NET_BIND_SERVICE)

### Network Options

- `--net-bridge <bridge>`: Specify the network bridge (default: docker0)
- `--net-ip <ip/prefix>`: Assign an IP address to the container (e.g., 172.17.0.2/24)
- `--port <host-port>:<container-port>/<protocol>`: Map ports from host to container

### Examples

1. Basic container with bash:
```bash
sudo ./container /bin:/usr/bin /root root /bin/bash xterm en_US.UTF-8 -- /bin/bash
```

2. Container with networking and port mapping:
```bash
sudo ./container /bin:/usr/bin /root root /bin/bash xterm en_US.UTF-8 \
  --net-bridge br0 --net-ip 172.17.0.2/24 --port 8080:80/tcp -- /bin/bash
```

3. Container with mount and capabilities:
```bash
sudo ./container /bin:/usr/bin /root root /bin/bash xterm en_US.UTF-8 \
  --mount /host/dir /container/dir rw \
  --cap-add CAP_NET_BIND_SERVICE \
  -- /bin/bash
```

## Notes

- You need root privileges to run the container as it requires creating namespaces and configuring networking.
- When specifying capabilities, use the full capability name (e.g., CAP_SYS_ADMIN).
- For security, it's recommended to drop all unnecessary capabilities.
- The network bridge must exist or have permissions to create it.

## Troubleshooting

- If you encounter networking issues, check if the bridge interface exists and iptables rules are properly set.
- For permission errors, ensure you're running with sudo or as root.
- Check the logs for detailed error messages.
