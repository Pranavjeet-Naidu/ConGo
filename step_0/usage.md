
# Basic Container in Go

This program demonstrates using Linux namespaces and cgroups to run a process in an isolated environment.

## Usage

1. Build and run:
   ```
   go run main_basic.go run <command> <args...>
   ```
2. Ensure the root filesystem is set up at /home/pj/ubuntufs or modify the path as needed.
3. The container will have its own process namespace, hostname, and mount namespace.

## Notes

• The cgroup limits the container to 20 processes.  
• For a minimal root filesystem, use debootstrap or Alpine as indicated in the code comments.