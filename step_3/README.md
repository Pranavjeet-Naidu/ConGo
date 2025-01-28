# step_3

This repository extends the container runtime from step_2 by adding user namespace and privilege dropping capabilities.

## new changes

1. **User Namespace Support**: Added `CLONE_NEWUSER` flag to completely isolate users inside the container
2. **UID/GID Mapping**: Added UID and GID mappings to map container root user to unprivileged host user
3. **Privilege Dropping**: Added capability to drop privileges to a specified user inside container
4. **Command Execution**: Modified command execution to handle user switching
5. **Configuration**: Extended Config struct to include user information
6. **Security Improvements**: Better isolation through user namespaces

## Key Technical Changes

1. Added to `Config` struct:
   ```go
   User string // New field for user namespace
   ```

2. Enhanced `cmd.SysProcAttr` with:
   ```go
   Cloneflags: // ...existing flags... | syscall.CLONE_NEWUSER,
   UidMappings: []syscall.SysProcIDMap{
       {
           ContainerID: 0,
           HostID:     os.Getuid(),
           Size:       1,
       },
   },
   GidMappings: []syscall.SysProcIDMap{
       {
           ContainerID: 0,
           HostID:     os.Getgid(),
           Size:       1,
       },
   },
   ```

3. New function `setupUser()` to handle privilege dropping

## Security Notes

- User namespaces provide an additional layer of security
- Container root user is mapped to unprivileged host user
- Better isolation between host and container processes
- Privilege dropping helps minimize potential security risks

## Retained Features from step_2

- Namespace isolation (UTS, PID, Network, Mount, IPC)
- Cgroups for resource limiting
- OverlayFS support
- Bind mounts
- Environment variable handling
- Resource limits (CPU, memory, processes)

## Building and Usage

See usage.md for detailed instructions on building and running the container runtime.
