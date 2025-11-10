# ConGo Internals

This document provides a high-level overview of the internal structure of the ConGo project. The core logic is organized into several packages within the `internals` directory.

## Directory Structure

```
internals/
├── capabilities/   # Manages Linux capabilities
├── cgroups/        # Cgroup management for resource control
├── config/         # Configuration parsing and validation
├── container/      # Core container lifecycle management
├── filesystem/     # Filesystem and rootfs setup
├── logging/        # Container logging
├── monitoring/     # Container monitoring
├── network/        # Container networking setup
├── setups/         # Initial container environment setup
├── state/          # Container state persistence
├── types/          # Common data types and constants
└── utils/          # Utility functions
```

## Package Overview

### `capabilities`

This package is responsible for managing Linux capabilities for the container process. It allows for fine-grained control over the privileges of the container, dropping unnecessary capabilities to enhance security.

### `cgroups`

The `cgroups` package handles the creation and management of control groups (cgroups) for containers. Cgroups are a Linux kernel feature used to limit, account for, and isolate the resource usage (CPU, memory, disk I/O, etc.) of a collection of processes. This package provides the logic to set resource limits like memory, CPU shares, and PID limits.

### `config`

This package manages all configuration-related tasks. It parses command-line arguments, validates the configuration, and creates the main `Config` struct that is passed around the application.

### `container`

This is the central package for managing the container lifecycle. It contains the logic for high-level operations such as:
- `run`, `create`, `start`, `stop`, `restart`, `rm`
- `exec`, `shell`
- `commit`, `logs`, `ps`
- `pause`, `unpause`
- Volume management

It orchestrates calls to other internal packages to perform these actions.

### `filesystem`

The `filesystem` package is responsible for setting up the container's root filesystem. This includes mounting the rootfs, setting up necessary directories like `/proc` and `/dev`, and handling volume mounts.

### `logging`

This package provides logging capabilities for containers. It captures the standard output and standard error streams of the container process and stores them in log files for later inspection with the `congo logs` command.

### `monitoring`

This package is intended for monitoring container resources. (Note: This may be a placeholder or under development).

### `network`

The `network` package handles setting up the network for the container. This can include creating network namespaces, setting up virtual Ethernet (veth) pairs, creating bridges, and managing IP addresses and port mappings.

### `setups`

The `setups` package is responsible for the initial environment setup inside the container, just before the user's command is executed. This includes setting the hostname, changing the root directory (`chroot`), mounting filesystems, and other initialization tasks that need to happen from within the new namespaces.

### `state`

This package manages the state of containers. It saves container configuration and status (e.g., "running", "stopped") to disk, typically as JSON files. This allows ConGo to manage containers across multiple commands and restarts.

### `types`

The `types` package defines the common data structures and constants used throughout the application. This includes the `Config` struct, `ContainerState`, and other important data types, ensuring consistency across different packages.

### `utils`

This package contains various utility functions that are used by other packages. This can include helper functions for string manipulation, user/group lookups, and wrappers for system calls.
