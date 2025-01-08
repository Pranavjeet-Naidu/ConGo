# Congo: A Step-by-Step Container Runtime Implementation

## About Congo
Congo is an educational project that implements a container runtime from scratch, helping developers understand how container technologies like Docker work under the hood. By building a container system piece by piece, this project demonstrates the core concepts of containerization, Linux namespaces, cgroups, and process isolation.Instead of being overwhelmed by the complexity of production container runtimes, Congo breaks down the implementation into manageable, educational pieces.

## Core Concepts
Congo demonstrates several fundamental concepts of containerization:
- Process Isolation using Linux namespaces
- Resource Management using cgroups
- Filesystem isolation using chroot and overlay filesystems
- Networking configuration and isolation
- Security and access control mechanisms

## Project Structure
The project is organized into multiple progressive steps, each in its own directory with detailed documentation:

Great! Let's incrementally enhance this program to make it more robust, feature-rich, and closer to modern containerization tools like Docker. Here’s a plan for building it up step-by-step:

---
### **Step 0: A Basic container**

- A basic implementation of a docker container using namespaces for isolation and cgroups for allocation of resources for this isolated environment . 


### **Step 1: Add Features for Flexibility**

- **Dynamic Root Filesystem**: Allow the user to specify the root filesystem (`chroot` directory) as a command-line argument.
- **Dynamic Resource Limits**: Let users set limits for processes, CPU, memory, etc., via command-line arguments.
- **Environment Variable Support**: Enable users to pass custom environment variables to the child process.

---

### **Step 2: Improve Isolation**

- **Network Namespace**: Use `CLONE_NEWNET` to isolate the network, enabling experiments with private network configurations.
- **IPC Namespace**: Add `CLONE_NEWIPC` to isolate interprocess communication.

---

### **Step 3: Add Storage Management**

- **Bind Mounting**: Allow mounting specific host directories into the container, mimicking Docker’s volume feature.
- **Layered Filesystem**: Explore union filesystems (e.g., OverlayFS) to simulate Docker image layering.

---

### **Step 4: Add User Management**

- **User Namespace**: Use `CLONE_NEWUSER` to map container users to non-root users on the host for enhanced security.
- **Capabilities**: Drop unnecessary Linux capabilities for the container process.

## Getting Started
Each step builds upon the previous ones, so it's recommended to follow them in order. Each directory contains:
- A detailed README explaining the concepts
- Implementation code with comments
- Examples and usage instructions
- Additional resources for learning

## Prerequisites
- Linux operating system (Ubuntu 20.04 or later recommended)
- Go programming language (1.16 or later)
- Basic understanding of Linux systems and containerization concepts
- Root access for namespace and cgroup operations

## Contributing
Contributions are welcome! Please read our contributing guidelines before submitting pull requests.

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Warning
This is an educational project and is not intended for production use. For production environments, please use established container runtimes like Docker or containerd.