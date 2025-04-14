# Step 4: Network Namespace and Container Networking

This repository extends the container runtime from step 3 by adding comprehensive network isolation and configuration capabilities.

## New Features

1. **Network Namespace Isolation**: Complete isolation of container networking with the `CLONE_NEWNET` flag
2. **Network Bridge Support**: Creates and configures network bridges to connect containers
3. **Virtual Ethernet Pairs**: Sets up veth pairs to connect containers to the host network
4. **IP Address Assignment**: Assigns IP addresses to containers
5. **Port Forwarding**: Maps ports from the host to the container
6. **Enhanced Capabilities Management**: More granular control over Linux capabilities

## Key Technical Changes

1. Added `NetworkConfig` struct:
   ```go
   type NetworkConfig struct {
       Bridge      string
       ContainerIP string
       PortMaps    []PortMapping
   }
   
   type PortMapping struct {
       HostPort      int
       ContainerPort int
       Protocol      string // "tcp" or "udp"
   }
   ```

2. New network setup functions:
   - `setupNetworking()`: Top-level network configuration
   - `createBridge()`: Creates bridge interfaces
   - `createVethPair()`: Creates virtual ethernet pairs
   - `connectToBridge()`: Connects veth interfaces to bridges
   - `setupContainerNetNS()`: Configures container network namespace
   - `setupPortForwarding()`: Configures port forwarding with iptables

3. Enhanced capability management:
   - `setupCapabilities()`: More granular control
   - `clearAllCapabilities()`: Drops all capabilities
   - `addCapability()`: Adds specific capabilities

## Networking Details

- Network namespaces provide complete network isolation
- Virtual ethernet pairs connect container to the host network
- Bridge interfaces connect multiple containers
- IP address assignment allows containers to communicate
- Port forwarding allows external access to container services

## Retained Features from Step 3

- User namespace isolation with UID/GID mapping
- Capability dropping for improved security
- Cgroups for resource limiting
- OverlayFS support for layered file systems
- Bind mounts for accessing host files
- Environment variable handling

## Security Considerations

- Network isolation prevents containers from interfering with host networking
- Capability management reduces the attack surface
- Port mappings allow controlled access to container services

## Building and Usage

See usage.md for detailed instructions on building and running the container runtime with networking features.
