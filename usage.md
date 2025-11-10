# Usage Guide

This guide provides detailed instructions on how to use `congo` to manage containers.

## Commands

### `run`

Create and start a new container in a single command.

**Usage:** `congo run [options] <image-path> <command> [args...]`

- **`--hostname <name>`**: Set the container's hostname.
- **`--memory <limit>`**: Set the memory limit (e.g., '100m', '1g').
- **`--cpu <shares>`**: Set the CPU shares (relative weight).
- **`--pids <limit>`**: Set the maximum number of PIDs.
- **`--interactive` or `-i`**: Run in interactive mode (starts a shell).
- **`--detached` or `-d`**: Run the container in the background.

**Example:**
```sh
sudo ./congo run --hostname my-container --memory 200m /path/to/rootfs /bin/echo "Hello, World!"
```

### `create`

Create a new container without starting it.

**Usage:** `congo create [options] <image-path> <command> [args...]`

**Example:**
```sh
sudo ./congo create --hostname my-container /path/to/rootfs /bin/sleep 100
```
This will output a container ID.

### `start`

Start a previously created container.

**Usage:** `congo start <container-id>`

**Example:**
```sh
sudo ./congo start congo-1678886400
```

### `ps`

List all containers.

**Usage:** `congo ps`

**Example:**
```sh
./congo ps
```

### `exec`

Execute a command inside a running container.

**Usage:** `congo exec <container-id> <command> [args...]`

**Example:**
```sh
sudo ./congo exec my-running-container /bin/ls -l /
```

### `shell`

Start an interactive shell (`/bin/bash` or `/bin/sh`) inside a running container.

**Usage:** `congo shell <container-id>`

**Example:**
```sh
sudo ./congo shell my-running-container
```

### `stop`

Stop a running container.

**Usage:** `congo stop <container-id> [--force]`

- **`--force`**: Force stop the container (sends SIGKILL).

**Example:**
```sh
sudo ./congo stop my-running-container
```

### `restart`

Restart a container.

**Usage:** `congo restart <container-id>`

**Example:**
```sh
sudo ./congo restart my-container
```

### `rm`

Remove a stopped container.

**Usage:** `congo rm <container-id>`

**Example:**
```sh
sudo ./congo rm my-stopped-container
```

### `logs`

Fetch the logs of a container.

**Usage:** `congo logs <container-id>`

**Example:**
```sh
./congo logs my-container
```

### `commit`

Create an image (a tarball of the rootfs) from a container's current state.

**Usage:** `congo commit <container-id> <image-name>`

**Example:**
```sh
sudo ./congo commit my-container my-custom-image
```

### `pause`

Pause all processes within a container.

**Usage:** `congo pause <container-id>`

**Example:**
```sh
sudo ./congo pause my-running-container
```

### `unpause`

Resume all processes within a paused container.

**Usage:** `congo unpause <container-id>`

**Example:**
```sh
sudo ./congo unpause my-paused-container
```

### `update`

Update the resource limits of a running container.

**Usage:** `congo update <container-id> [--memory=<limit>] [--cpu=<shares>] [--pids=<limit>]`

**Example:**
```sh
sudo ./congo update my-running-container --memory 512m --cpu 2048
```

### Volume Management

#### `volume-add`

Add a volume mount to a running container.

**Usage:** `congo volume-add <container-id> <host-path> <container-path> [ro]`

- **`ro`**: Mount the volume as read-only.

**Example:**
```sh
sudo ./congo volume-add my-container /data/shared /mnt/shared
```

#### `volume-remove`

Remove a volume mount from a container.

**Usage:** `congo volume-remove <container-id> <container-path>`

**Example:**
```sh
sudo ./congo volume-remove my-container /mnt/shared
```
