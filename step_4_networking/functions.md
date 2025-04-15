## Network-related Functions

### setupNetworking(config *Config) error
- Creates a network bridge if it doesn't exist
- Creates a virtual ethernet (veth) pair
- Connects the host veth to the bridge
- Sets up the container network namespace
- Configures port forwarding

### createBridge(name string) error
- Checks if the specified bridge exists
- Creates a new bridge interface using the ip command
- Sets the bridge interface up

### createVethPair(container, host string) error
- Creates a veth pair with specified names
- Sets the host interface up

### connectToBridge(veth, bridge string) error
- Connects a veth interface to the specified bridge

### setupContainerNetNS(veth, ip string) error
- Moves the veth interface to the container namespace
- Sets up the loopback interface
- Configures the container veth interface
- Assigns an IP address to the container veth

### setupPortForwarding(portMaps []PortMapping, containerIP string) error
- Adds iptables rules for port forwarding
- Maps host ports to container ports

## Capability Management Functions

### setupCapabilities(config *Config) error
- Maps capability names to their numeric values
- Drops all capabilities by default
- Adds back only specified capabilities

### clearAllCapabilities() error
- Clears all ambient capabilities
- Drops capabilities from the bounding set

### addCapability(capValue uintptr) error
- Adds a capability to the ambient set
- Sets PR_SET_KEEPCAPS to retain capabilities
- Adds capability to the permitted and effective sets

## Configuration Functions

### parseConfig(args []string, isChild bool) (*Config, error)
- Parses command-line arguments
- Sets environment variables
- Handles mount, user, and capability options
- Extracts the command to execute

### validateConfig(config *Config) error
- Validates the configuration settings
- Checks if required tools are available

## Container Setup Functions

### setupContainer(config *Config) error
- Sets up the hostname
- Configures the root filesystem
- Sets up capabilities
- Configures bind mounts
- Sets up cgroups
- Sets up user namespace
- Configures environment variables

### setupLayeredRootfs(config *Config) error
- Creates the work and upper directories for OverlayFS
- Mounts the overlay filesystem using specified layers

### setupRootfs(rootfs string) error
- Makes the root filesystem private
- Binds the rootfs to itself
- Changes directory to rootfs
- Mounts the proc filesystem

### setupMounts(mounts []Mount) error
- Creates mount points
- Performs bind mounts with read-only option if specified

### setupCgroups(config *Config) error
- Creates cgroup directories
- Sets process limits
- Configures memory limits
- Sets CPU shares

## Utility Functions

### cleanup(config *Config) error
- Removes cgroup directories
- Unmounts filesystems

### parseEnvVars(envStr string) map[string]string
- Parses environment variables from a string

### formatEnvVars(envMap map[string]string) string
- Formats environment variables into a string

### parseMountSpec(spec string) (Mount, error)
- Parses mount specifications from command-line arguments

### mustAtoi(s string) int
- Converts a string to an integer or fails fatally
