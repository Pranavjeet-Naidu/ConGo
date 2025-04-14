# Usage Instructions

## Prerequisites

1. Ensure you have Go installed on your system.
2. Create a root filesystem directory and populate it with the root filesystem. You can use `debootstrap` or `alpine linux` for this purpose.

### Using debootstrap:
```sh
sudo apt-get install debootstrap 
sudo debootstrap stable /home/pj/ubuntufs http://deb.debian.org/debian/
```

### Using Alpine Linux:
```sh
wget https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-latest-x86_64.tar.gz
sudo tar -xzf alpine-minirootfs-latest-x86_64.tar.gz -C /home/pj/ubuntufs
```

## Building

Make sure you have Go installed. Then run:

```bash
cd /home/grass/projects/congo/step_2
go build -o container-runtime main.go
```

## Running the Program

Run the following command to create a new container environment:

```bash
go run main.go run <rootfs> <process_limit> <memory_limit> <cpu_share> <env_vars> [--mount source:dest[:ro]] [--layers layer1,layer2] -- <cmd> <args>
```

Where:
- `<rootfs>` is the root filesystem for the container.
- `<process_limit>` is the process limit.
- `<memory_limit>` is the memory limit (e.g., `512m`).
- `<cpu_share>` is the CPU share.
- `<env_vars>` specifies environment variables (e.g., `KEY=VALUE,FOO=BAR`).
- `[--mount source:dest[:ro]]` specifies bind mounts (optional).
- `[--layers layer1,layer2]` specifies OverlayFS layers (optional).
- `<cmd>` is the program to execute inside the container.
- `<args>` are the arguments for the command.

### Example

To run `bash` inside the container with specific limits, use:

```bash
go run main.go run /home/pj/ubuntufs 100 512m 1024 KEY=VALUE -- /bin/bash
```

To run `bash` inside the container with bind mounts and OverlayFS layers, use:

```bash
go run main.go run /home/pj/ubuntufs 100 512m 1024 KEY=VALUE --mount /host/path:/container/path:ro --layers /layer1,/layer2 -- /bin/bash
```

## Notes

- **Security:** This is a simplified demonstration and may not handle all security concerns of a production container runtime.
- **Privileges:** Running this code requires root privileges because it sets up namespaces and cgroups.

## Troubleshooting

- Check logs for messages prefixed with `container-runtime:` if something fails.
- Ensure your kernel supports required namespaces and cgroup controllers.