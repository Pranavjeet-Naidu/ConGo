

##  Container Lifecycle Commands

### Creating a Container

Create a new container without starting it:

```bash
sudo ./main create [standard_args] --id my-container -- /bin/sh
```

This creates a container with the specified ID and configuration but doesn't start it.

### Starting a Container

Start a previously created container:

```bash
sudo ./main start my-container
```

You can also override the command at start time:

```bash
sudo ./main start my-container /bin/bash
```

### Stopping a Container

Stop a running container gracefully:

```bash
sudo ./main stop my-container
```

For forceful termination:

```bash
sudo ./main stop my-container --force
```

### Removing a Container

Remove a stopped container:

```bash
sudo ./main rm my-container
```

### Single-step Create and Run

Create and start a container in a single command:

```bash
sudo ./main run [standard_args] --id my-container -- /bin/sh
```

## Container Operation Modes

### Interactive Mode

Run a container with interactive shell:

```bash
sudo ./main run [standard_args] --interactive -- /bin/bash
```

Or use the short form:

```bash
sudo ./main run [standard_args] -i -- /bin/bash
```

### Detached Mode

Run a container in the background:

```bash
sudo ./main run [standard_args] --detach -- /bin/sh -c "while true; do date; sleep 1; done"
```

Or use the short form:

```bash
sudo ./main run [standard_args] -d -- /bin/sh -c "while true; do date; sleep 1; done"
```

## Container Resource Management

### Updating Container Resources

Update memory limit of a running container:

```bash
sudo ./main update my-container --memory=512M
```

Update CPU shares:

```bash
sudo ./main update my-container --cpu=1024
```

Update process limit:

```bash
sudo ./main update my-container --pids=100
```

Multiple updates at once:

```bash
sudo ./main update my-container --memory=1G --cpu=2048 --pids=200
```

## Managing Container State

### Pausing and Unpausing Containers

Pause a running container:

```bash
sudo ./main pause my-container
```

Resume a paused container:

```bash
sudo ./main unpause my-container
```

### Executing Commands in Running Containers

Execute a command in a running container:

```bash
sudo ./main exec my-container ls -la
```

Start an interactive shell in a running container:

```bash
sudo ./main shell my-container
```

### Container Commit

Create an image from a container:

```bash
sudo ./main commit my-container my-new-image
```

## Volume Management

### Adding Volumes to Running Containers

Add a read-write volume:

```bash
sudo ./main volume-add my-container /host/path /container/path
```

Add a read-only volume:

```bash
sudo ./main volume-add my-container /host/path /container/path ro
```

### Removing Volumes from Running Containers

```bash
sudo ./main volume-remove my-container /container/path
```

## Container Information

### Listing Containers

List all containers:

```bash
sudo ./main ps
```

This shows container IDs, status, creation time, and commands.

### Viewing Container Logs

View container logs:

```bash
sudo ./main logs my-container
```

## Combining with Monitoring

Use container lifecycle features with monitoring:

```bash
sudo ./main run [standard_args] --enable-monitor --monitor-interval 5 -d -- /bin/sh -c "while true; do echo 'Running'; sleep 5; done"
```

Then view the stats:

```bash
sudo ./main logs my-container
```

## Troubleshooting

If you encounter issues with container lifecycle operations:

1. Check container status with `sudo ./main ps`
2. Verify the container ID is correct
3. Ensure you have sufficient permissions for the operation
4. Check logs for detailed error messages
5. For stuck containers, use `--force` flag with stop command
