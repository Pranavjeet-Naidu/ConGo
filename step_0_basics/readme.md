# Basic Container in Go

This project demonstrates how to use Linux namespaces and cgroups to run a process in an isolated environment using Go.

## Features

- Isolates processes using Linux namespaces (UTS, PID, mount).
- Limits resources using cgroups.
- Provides a minimal container runtime.

## Prerequisites

- Go installed on your system.
- A root filesystem directory populated with a minimal Linux distribution (e.g., using debootstrap or Alpine Linux).

## Usage

1. Create a root filesystem directory and populate it with the root filesystem.
2. Navigate to the directory containing `main_basic.go`.
3. Run the following command to execute the program:

```sh
go run main_basic.go run <cmd> <args>
```

Replace `<cmd>` with the command you want to run inside the container and `<args>` with the arguments for the command.

### Example

To run `bash` inside the container, use:

```sh
go run main_basic.go run /bin/bash
```

## Notes

- The cgroup limits the container to 20 processes.
- For a minimal root filesystem, use debootstrap or Alpine as indicated in the code comments.

## License

This project is licensed under the MIT License.
