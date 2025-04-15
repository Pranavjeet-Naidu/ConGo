# Congo: Container Lifecycle Management Functions

This document describes the key functions related to the container lifecycle management functionality in step 6 of the Congo container implementation.

## Container State Management

### `saveContainerState`

```go
func saveContainerState(containerID string, state ContainerState) error
```

Persists container state to disk:
- Serializes the container state to JSON
- Writes the state to a file named after the container ID
- Ensures the state directory exists

### `loadContainerState`

```go
func loadContainerState(containerID string) (ContainerState, error)
```

Retrieves container state from disk:
- Reads the state file for the specified container ID
- Deserializes the JSON data into a ContainerState struct
- Returns the container state for further operations

### `listContainers`

```go
func listContainers() ([]ContainerState, error)
```

Lists all containers in the system:
- Reads all state files from the container state directory
- Parses each file to extract container information
- Returns a list of all container states

## Container Lifecycle Operations

### `startContainer`

```go
func startContainer(containerID string, args []string) error
```

Starts an existing container:
- Loads the container state
- Verifies the container isn't already running
- Creates the container process using the stored configuration
- Updates the container state to "running"

### `stopContainer`

```go
func stopContainer(containerID string, force bool) error
```

Stops a running container:
- Sends SIGTERM to the container (or SIGKILL if force=true)
- Waits for container to exit with timeout
- Cleans up resources like network interfaces
- Updates container state to "stopped"

### `pauseContainer`

```go
func pauseContainer(containerID string) error
```

Pauses a running container:
- Uses cgroup freezer to suspend container processes
- Updates container state to "paused"
- Preserves all resources while container is paused

### `unpauseContainer`

```go
func unpauseContainer(containerID string) error
```

Unpauses a paused container:
- Thaws the container processes in the freezer cgroup
- Updates container state to "running"
- Container continues execution from where it was paused

### `removeContainer`

```go
func removeContainer(containerID string) error
```

Removes a container:
- Ensures container is not running
- Deletes container state file
- Cleans up container resources (logs, etc.)

## Advanced Container Management

### `commitContainer`

```go
func commitContainer(containerID, imageName string) error
```

Creates an image from a container:
- Creates a tarball of the container filesystem
- Saves container metadata (command, environment, etc.)
- Stores the image for future container creation

### `execInContainer`

```go
func execInContainer(containerID string, command []string) error
```

Executes a command in a running container:
- Uses nsenter to run a process in the container's namespaces
- Connects to container's stdin/stdout/stderr
- Preserves container isolation

### `updateContainerResources`

```go
func updateContainerResources(containerID, memory, cpu string, pids int) error
```

Updates resource limits for a running container:
- Modifies cgroup parameters for memory, CPU, and process limits
- Updates container state to reflect new limits
- Takes effect immediately on the running container

### `addVolumeToContainer`

```go
func addVolumeToContainer(containerID, hostPath, containerPath string, readOnly bool) error
```

Adds a new volume to a running container:
- Creates the mount point inside the container
- Performs the bind mount operation
- Updates container state with new mount information

### `removeVolumeFromContainer`

```go
func removeVolumeFromContainer(containerID, containerPath string) error
```

Removes a volume from a running container:
- Unmounts the volume from the container
- Updates container state to remove mount information
