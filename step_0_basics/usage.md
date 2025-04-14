# Basic Container in Go

This program demonstrates using Linux namespaces and cgroups to run a process in an isolated environment.

## Usage Instructions

### Prerequisites

1. Ensure you have Go installed on your system.
2. Create a root filesystem directory and populate it with the root filesystem. You can use `debootstrap` or `alpine linux` for this purpose.

#### Using debootstrap:
```sh
sudo apt-get install debootstrap 
sudo debootstrap stable /home/pj/ubuntufs http://deb.debian.org/debian/
```

#### Using Alpine Linux:
```sh
wget https://dl-cdn.alpinelinux.org/alpine/latest-stable/releases/x86_64/alpine-minirootfs-latest-x86_64.tar.gz
sudo tar -xzf alpine-minirootfs-latest-x86_64.tar.gz -C /home/pj/ubuntufs
```

### Running the Program

1. Navigate to the directory containing `main_basic.go`.
2. Run the following command to execute the program:

```sh
go run main_basic.go run <cmd> <args>
```

Replace `<cmd>` with the command you want to run inside the container and `<args>` with the arguments for the command.

#### Example

To run `bash` inside the container, use:

```sh
go run main_basic.go run /bin/bash
```

This will create a new namespace, isolate it from the host, and run the specified command inside the container.

## Notes

• The cgroup limits the container to 20 processes.  
• For a minimal root filesystem, use debootstrap or Alpine as indicated in the code comments.