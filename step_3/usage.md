# Usage Instructions

## Prerequisites

1. Ensure you have Go installed on your system
2. Ensure your kernel supports user namespaces
3. Create a root filesystem directory (using debootstrap or Alpine Linux)

## Building

```bash
cd /home/grass/projects/congo/step_3
go build -o container-runtime main.go
```

## Running Containers

Basic syntax:
```bash
sudo ./container-runtime run <rootfs> <process_limit> <memory_limit> <cpu_share> <env_vars> [--user username] [--mount source:dest[:ro]] [--layers layer1,layer2] -- <cmd> <args>
```

### New User Namespace Features

Run container as a specific user:
```bash
sudo ./container-runtime run /path/to/rootfs 100 512m 1024 PATH=/usr/bin --user unprivileged_user -- /bin/bash
```

### Examples

1. Basic container with user namespace:
```bash
sudo ./container-runtime run /path/to/rootfs 100 512m 1024 PATH=/usr/bin --user nobody -- /bin/sh
```

2. Container with mounts and user:
```bash
sudo ./container-runtime run /path/to/rootfs 100 512m 1024 PATH=/usr/bin --user nobody --mount /host/path:/container/path:ro -- /bin/sh
```

## Environment Variables

Default environment variables set:
- PATH
- HOME
- USER
- SHELL
- TERM
- LANG

## Security Considerations

1. User namespace isolation is active by default
2. Container root is mapped to unprivileged host user
3. Additional privilege dropping through --user flag
4. Mount points are properly isolated through namespaces

## Troubleshooting

1. Check kernel support for user namespaces:
```bash
cat /proc/sys/kernel/unprivileged_userns_clone
```

2. Verify user exists before using --user flag
3. Check logs for namespace or permission errors
