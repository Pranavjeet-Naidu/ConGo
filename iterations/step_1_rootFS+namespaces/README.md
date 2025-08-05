
provides a simple container runtime written in Go. It leverages Linux namespaces, cgroups, and chroot to isolate processes.

## New Changes 

- **Additional Namespaces:** Added support for Network and IPC namespaces.
- **Configuration Parsing:** Introduced `parseConfig` to handle command-line arguments and create a configuration struct.
- **Environment Variables:** Added support for setting environment variables inside the container.
- **Resource Limits:** Enhanced cgroup setup to include memory and CPU limits.
- **Root Filesystem Setup:** Improved root filesystem setup with `setupRootfs` function.
- **Validation:** Added `validateConfig` to ensure configuration parameters are valid.

## Features

- Isolates processes using Linux namespaces (UTS, PID, Network, Mount, IPC).
- Limits resources using cgroups.
- Sets up a new root filesystem and mounts essential directories.


## License

This project is licensed under the MIT License.