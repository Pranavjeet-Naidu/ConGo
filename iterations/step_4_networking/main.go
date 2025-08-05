package main

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
    "golang.org/x/sys/unix"
    "net"
)

type Mount struct {
    Source      string
    Destination string
    ReadOnly    bool
}

type NetworkConfig struct {
    Bridge      string
    ContainerIP string
    PortMaps    []PortMapping
}

// adding some constants that are not defined in unix package
const (
    PR_CAP_PERMITTED  = 2
    PR_CAP_EFFECTIVE  = 3
    PR_CAP_INHERITABLE = 1
)
// Config stores container configuration
type Config struct {
    Rootfs       string
    ProcessLimit int
    MemoryLimit  string
    CpuShare     string
    EnvVars      map[string]string
    Command      []string
    Mounts       []Mount
    UseLayers    bool     
    ImageLayers  []string 
    User         string   // New field for user namespace
    Capabilities []string // New field for capabilities
	Network NetworkConfig
}

type PortMapping struct {
    HostPort      int
    ContainerPort int
    Protocol      string // "tcp" or "udp"
}



func setupCapabilities(config *Config) error {
    // Map of capability names to their numeric values
    capMap := map[string]uintptr{
        "CAP_CHOWN":            0,
        "CAP_DAC_OVERRIDE":     1,
        "CAP_DAC_READ_SEARCH":  2,
        "CAP_FOWNER":           3,
        "CAP_FSETID":           4,
        "CAP_KILL":             5,
        "CAP_SETGID":           6,
        "CAP_SETUID":           7,
        "CAP_SETPCAP":          8,
        "CAP_NET_BIND_SERVICE": 10,
        "CAP_NET_RAW":          13,
        "CAP_SYS_CHROOT":       18,
        "CAP_MKNOD":            27,
        "CAP_AUDIT_WRITE":      29,
        "CAP_SETFCAP":          31,
    }


    if len(config.Capabilities) == 0 {
        // Drop all capabilities by default
        log.Println("Dropping all capabilities")
        if err := clearAllCapabilities(); err != nil {
            return fmt.Errorf("failed to clear all capabilities: %v", err)
        }
        return nil
    }

    // Keep only specified capabilities
    log.Printf("Setting up capabilities: %v", config.Capabilities)
    
    // First drop all capabilities
    if err := clearAllCapabilities(); err != nil {
        return fmt.Errorf("failed to clear all capabilities: %v", err)
    }
    
    // Then add back the ones specified
    for _, cap := range config.Capabilities {
        capValue, exists := capMap[cap]
        if !exists {
            return fmt.Errorf("unknown capability: %s", cap)
        }
        
        if err := addCapability(capValue); err != nil {
            return fmt.Errorf("failed to add capability %s: %v", cap, err)
        }
        log.Printf("Added capability: %s", cap)
    }
    
    return nil
}

func clearAllCapabilities() error {
    // Clear all ambient capabilities
    if err := unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0); err != nil {
        return fmt.Errorf("failed to clear ambient capabilities: %v", err)
    }
    
    // Clear bounding set capabilities
    for i := uintptr(0); i <= 40; i++ { // Loop through all possible capability values
        unix.Prctl(unix.PR_CAPBSET_DROP, i, 0, 0, 0)
    }
    
    return nil
}

func addCapability(capValue uintptr) error {
    // Set capability in the permitted set
    if err := unix.Prctl(unix.PR_CAP_AMBIENT, unix.PR_CAP_AMBIENT_RAISE, capValue, 0, 0); err != nil {
        return fmt.Errorf("failed to add capability to ambient set: %v", err)
    }
    
    // Set capability in the permitted set using direct value
    if err := unix.Prctl(unix.PR_SET_KEEPCAPS, 1, 0, 0, 0); err != nil {
        return fmt.Errorf("failed to set PR_SET_KEEPCAPS: %v", err)
    }
    
    // Set capability in the permitted set
    if err := unix.Prctl(PR_CAP_PERMITTED, 1, capValue, 0, 0); err != nil {
        return fmt.Errorf("failed to add capability to permitted set: %v", err)
    }
    
    // Set capability in the effective set
    if err := unix.Prctl(PR_CAP_EFFECTIVE, 1, capValue, 0, 0); err != nil {
        return fmt.Errorf("failed to add capability to effective set: %v", err)
    }
    
    return nil
}

func parseConfig(args []string, isChild bool) (*Config, error) {
    config := &Config{
        EnvVars: make(map[string]string),
        Mounts:  make([]Mount, 0),
        Capabilities: make([]string, 0),
        Network: NetworkConfig{
            Bridge:      "congo0",  // Default bridge name
            PortMaps:    make([]PortMapping, 0),
        },
    }
    if len(args) < 7 {
        return nil, fmt.Errorf("not enough arguments")
    }

    config.EnvVars["PATH"] = args[1]
    config.EnvVars["HOME"] = args[2]
    config.EnvVars["USER"] = args[3]
    config.EnvVars["SHELL"] = args[4]
    config.EnvVars["TERM"] = args[5]
    config.EnvVars["LANG"] = args[6]

    cmdIndex := 7
    for i, arg := range args {
        if arg == "--" {
            cmdIndex = i
            break
        }
    }

    if cmdIndex == len(args) {
        return nil, fmt.Errorf("no command specified")
    }

    // Parse additional arguments before --
    currentIdx := 7
    for currentIdx < cmdIndex {
        switch args[currentIdx] {
        case "--mount":
            if currentIdx+3 >= cmdIndex {
                return nil, fmt.Errorf("missing mount specification")
            }
            mount := Mount{
                Source:      args[currentIdx+1],
                Destination: args[currentIdx+2],
                ReadOnly:    args[currentIdx+3] == "ro",
            }
            config.Mounts = append(config.Mounts, mount)
            currentIdx += 4
        case "--user":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing user specification")
            }
            config.User = args[currentIdx+1]
            currentIdx += 2
        case "--cap-add":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing capability specification")
            }
            config.Capabilities = append(config.Capabilities, args[currentIdx+1])
            currentIdx += 2
        case "--net-bridge":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing bridge name")
            }
            config.Network.Bridge = args[currentIdx+1]
            currentIdx += 2
        case "--net-ip":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing IP address")
            }
            config.Network.ContainerIP = args[currentIdx+1]
            currentIdx += 2
        case "--port":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing port mapping")
            }
            portMap, err := parsePortMapping(args[currentIdx+1])
            if err != nil {
                return nil, fmt.Errorf("invalid port mapping: %v", err)
            }
            config.Network.PortMaps = append(config.Network.PortMaps, portMap)
            currentIdx += 2
        default:
            return nil, fmt.Errorf("unknown option: %s", args[currentIdx])
        }
    }

    config.Command = args[cmdIndex+1:]
    return config, nil
}

func setupContainer(config *Config) error {
    defer cleanup(config)

    // Set hostname
    if err := syscall.Sethostname([]byte("container")); err != nil {
        return fmt.Errorf("error setting hostname: %v", err)
    }

    // Setup root filesystem
    if config.UseLayers {
        if err := setupLayeredRootfs(config); err != nil {
            return fmt.Errorf("error setting up layered rootfs: %v", err)
        }
    } else {
        if err := setupRootfs(config.Rootfs); err != nil {
            return fmt.Errorf("error setting up rootfs: %v", err)
        }
    }
     // Add capability setup early in the process
     if err := setupCapabilities(config); err != nil {
        return fmt.Errorf("error setting up capabilities: %v", err)
    }

    // Setup bind mounts
    if err := setupMounts(config.Mounts); err != nil {
        return fmt.Errorf("error performing bind mounts: %v", err)
    }

    // Setup cgroups
    if err := setupCgroups(config); err != nil {
            return fmt.Errorf("error setting up cgroups: %v", err)
    }

    // Setup user (new functionality)
    if config.User != "" {
        if err := setupUser(config.User); err != nil {
            return fmt.Errorf("error setting up user: %v", err)
        }
    }

    // Setup environment variables
    for k, v := range config.EnvVars {
        if err := os.Setenv(k, v); err != nil {
            return fmt.Errorf("error setting environment variable %s: %v", k, err)
        }
    }
        // Setup networking if IP is specified
    if config.Network.ContainerIP != "" {
        if err := setupNetworking(config); err != nil {
            return fmt.Errorf("error setting up networking: %v", err)
        }
    }

    return nil
}

func setupUser(user string) error {
    // Drop privileges to the specified user
    cmd := exec.Command("su", "-", user)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to switch user: %v", err)
    }

    return nil
}

// Add these new functions

func setupNetworking(config *Config) error {
    // Create bridge if it doesn't exist
    if err := createBridge(config.Network.Bridge); err != nil {
        return fmt.Errorf("failed to create bridge: %v", err)
    }

    // Create veth pair
    containerVeth := fmt.Sprintf("veth%d", os.Getpid())
    hostVeth := fmt.Sprintf("hveth%d", os.Getpid())

    if err := createVethPair(containerVeth, hostVeth); err != nil {
        return fmt.Errorf("failed to create veth pair: %v", err)
    }

    // Connect host veth to bridge
    if err := connectToBridge(hostVeth, config.Network.Bridge); err != nil {
        return fmt.Errorf("failed to connect to bridge: %v", err)
    }

    // Setup container network namespace
    if err := setupContainerNetNS(containerVeth, config.Network.ContainerIP); err != nil {
        return fmt.Errorf("failed to setup container network namespace: %v", err)
    }

    // Setup port forwarding
    if err := setupPortForwarding(config.Network.PortMaps, config.Network.ContainerIP); err != nil {
        return fmt.Errorf("failed to setup port forwarding: %v", err)
    }

    return nil
}

func createBridge(name string) error {
    // Check if bridge exists
    if _, err := net.InterfaceByName(name); err == nil {
        return nil // Bridge already exists
    }

    // Create bridge using ip command
    if err := exec.Command("ip", "link", "add", name, "type", "bridge").Run(); err != nil {
        return err
    }

    // Set bridge up
    if err := exec.Command("ip", "link", "set", name, "up").Run(); err != nil {
        return err
    }

    return nil
}

func createVethPair(container, host string) error {
    // Create veth pair
    if err := exec.Command("ip", "link", "add", container, "type", "veth", "peer", "name", host).Run(); err != nil {
        return err
    }

    // Set host interface up
    if err := exec.Command("ip", "link", "set", host, "up").Run(); err != nil {
        return err
    }

    return nil
}

func connectToBridge(veth, bridge string) error {
    return exec.Command("ip", "link", "set", veth, "master", bridge).Run()
}

func setupContainerNetNS(veth, ip string) error {
    // Move veth to container namespace
    if err := exec.Command("ip", "link", "set", veth, "netns", strconv.Itoa(os.Getpid())).Run(); err != nil {
        return err
    }

    // Setup loopback interface
    if err := exec.Command("ip", "link", "set", "lo", "up").Run(); err != nil {
        return err
    }

    // Setup container veth
    if err := exec.Command("ip", "link", "set", veth, "up").Run(); err != nil {
        return err
    }

    // Assign IP to container veth
    if err := exec.Command("ip", "addr", "add", ip, "dev", veth).Run(); err != nil {
        return err
    }

    return nil
}

func setupPortForwarding(portMaps []PortMapping, containerIP string) error {
    for _, port := range portMaps {
        // Add iptables DNAT rule for port forwarding
        rule := fmt.Sprintf(
            "-t nat -A PREROUTING -p %s --dport %d -j DNAT --to-destination %s:%d",
            port.Protocol,
            port.HostPort,
            containerIP,
            port.ContainerPort,
        )
        
        if err := exec.Command("iptables", strings.Split(rule, " ")...).Run(); err != nil {
            return fmt.Errorf("failed to add port forwarding rule: %v", err)
        }
    }
    return nil
}

func setupMounts(mounts []Mount) error {
    for _, mount := range mounts {
        if err := os.MkdirAll(mount.Destination, 0755); err != nil {
            return fmt.Errorf("failed to create mount point: %v", err)
        }

        flags := syscall.MS_BIND
        if mount.ReadOnly {
            flags |= syscall.MS_RDONLY
        }

        if err := syscall.Mount(mount.Source, mount.Destination, "", uintptr(flags), ""); err != nil {
            return fmt.Errorf("failed to mount: %v", err)
        }
    }
    return nil
}

func setupEnv(envVars map[string]string) error {
    for key, value := range envVars {
        if err := os.Setenv(key, value); err != nil {
            return fmt.Errorf("failed to set environment variable: %v", err)
        }
    }
    return nil
}

func cleanup(config *Config) error {
    log.Println("Cleaning up cgroups and unmounts")

    // Remove cgroup directories
    containerId := fmt.Sprintf("container-%d", os.Getpid())
    cgroupPaths := []string{
        filepath.Join("/sys/fs/cgroup/pids", containerId),
        filepath.Join("/sys/fs/cgroup/memory", containerId),
        filepath.Join("/sys/fs/cgroup/cpu", containerId),
        filepath.Join("/sys/fs/cgroup/blkio", containerId),
    }

    for _, path := range cgroupPaths {
        if err := os.RemoveAll(path); err != nil {
            return fmt.Errorf("failed to remove cgroup path %s: %v", path, err)
        }
    }

    for _, mount := range config.Mounts {
        if err := syscall.Unmount(mount.Destination, 0); err != nil {
            return fmt.Errorf("failed to unmount: %v", err)
        }
    }
    return nil
}

func validateConfig(config *Config) error {
    if config == nil {
        return fmt.Errorf("config cannot be nil")
    }

    if config.User != "" {
        if _, err := exec.LookPath("su"); err != nil {
            return fmt.Errorf("su command not found, required for user switching")
        }
    }

    return nil
}

func setupCgroups(config *Config) error {
    containerId := fmt.Sprintf("container-%d", os.Getpid())
    cgroupPaths := map[string]string{
        "pids":   filepath.Join("/sys/fs/cgroup/pids", containerId),
        "memory": filepath.Join("/sys/fs/cgroup/memory", containerId),
        "cpu":    filepath.Join("/sys/fs/cgroup/cpu", containerId),
        "blkio":  filepath.Join("/sys/fs/cgroup/blkio", containerId),
    }

    for subsystem, path := range cgroupPaths {
        if err := os.MkdirAll(path, 0755); err != nil {
            return fmt.Errorf("failed to create cgroup path %s: %v", path, err)
        }

        switch subsystem {
        case "pids":
            if err := os.WriteFile(filepath.Join(path, "pids.max"), []byte(strconv.Itoa(config.ProcessLimit)), 0644); err != nil {
                return fmt.Errorf("failed to set pids.max: %v", err)
            }
            if err := os.WriteFile(filepath.Join(path, "notify_on_release"), []byte("1"), 0644); err != nil {
                return fmt.Errorf("failed to set notify_on_release: %v", err)
            }
        case "memory":
            if err := os.WriteFile(filepath.Join(path, "memory.limit_in_bytes"), []byte(config.MemoryLimit), 0644); err != nil {
                return fmt.Errorf("failed to set memory.limit_in_bytes: %v", err)
            }
        case "cpu":
            if err := os.WriteFile(filepath.Join(path, "cpu.shares"), []byte(config.CpuShare), 0644); err != nil {
                return fmt.Errorf("failed to set cpu.shares: %v", err)
            }
        case "blkio":
            // Add blkio settings if needed
        }
    }

    return nil
}

func setupLayeredRootfs(config *Config) error {
    // Create work and upper directories
    workDir := filepath.Join(os.TempDir(), "overlay-work")
    upperDir := config.Rootfs

    if err := os.MkdirAll(workDir, 0755); err != nil {
        return fmt.Errorf("failed to create work directory: %v", err)
    }

    if err := os.MkdirAll(upperDir, 0755); err != nil {
        return fmt.Errorf("failed to create upper directory: %v", err)
    }

    lowerDirs := strings.Join(config.ImageLayers, ":")
    overlayOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDirs, upperDir, workDir)

    if err := syscall.Mount("overlay", "/", "overlay", 0, overlayOpts); err != nil {
        return fmt.Errorf("failed to mount overlay filesystem: %v", err)
    }

    return nil
}

func setupRootfs(rootfs string) error {
    if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
        return fmt.Errorf("error making root private: %v", err)
    }

    if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
        return fmt.Errorf("error binding rootfs: %v", err)
    }

    if err := os.Chdir(rootfs); err != nil {
        return fmt.Errorf("error changing directory to rootfs: %v", err)
    }

    if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
        return fmt.Errorf("error mounting proc: %v", err)
    }

    return nil
}

func parseEnvVars(envStr string) map[string]string {
    envVars := make(map[string]string)
    pairs := strings.Split(envStr, ",")
    for _, pair := range pairs {
        kv := strings.SplitN(pair, "=", 2)
        if len(kv) == 2 {
            envVars[kv[0]] = kv[1]
        }
    }
    return envVars
}

func formatEnvVars(envMap map[string]string) string {
    var envStrs []string
    for k, v := range envMap {
        envStrs = append(envStrs, fmt.Sprintf("%s=%s", k, v))
    }
    return strings.Join(envStrs, ",")
}

func main() {
    if len(os.Args) < 2 {
        log.Fatalf("Usage: %s <command> [args...]", filepath.Base(os.Args[0]))
    }

    isChild := os.Args[1] == "child"
    config, err := parseConfig(os.Args[1:], isChild)
    if err != nil {
        log.Fatalf("Error parsing config: %v", err)
    }

    if err := validateConfig(config); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    if isChild {
        if err := setupContainer(config); err != nil {
            log.Fatalf("Error setting up container: %v", err)
        }

        if err := syscall.Exec(config.Command[0], config.Command, os.Environ()); err != nil {
            log.Fatalf("Error executing command: %v", err)
        }
    } else {
        cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[1:]...)...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Cloneflags: syscall.CLONE_NEWUTS |
                syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS |
                syscall.CLONE_NEWNET |
                syscall.CLONE_NEWIPC |
                syscall.CLONE_NEWUSER,  // Add user namespace
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
            Unshareflags: syscall.CLONE_NEWNS,
        }

        if err := cmd.Run(); err != nil {
            log.Fatalf("Error running child process: %v", err)
        }
    }
}

func parseMountSpec(spec string) (Mount, error) {
    parts := strings.Split(spec, ":")
    if len(parts) < 2 || len(parts) > 3 {
        return Mount{}, fmt.Errorf("invalid mount specification: %s", spec)
    }

    mount := Mount{
        Source:      parts[0],
        Destination: parts[1],
        ReadOnly:    len(parts) == 3 && parts[2] == "ro",
    }

    return mount, nil
}

func mustAtoi(s string) int {
    i, err := strconv.Atoi(s)
    if (err != nil) {
        log.Fatalf("Invalid number %s: %v", s, err)
    }
    return i
}

func parsePortMapping(spec string) (PortMapping, error) {
    // Format should be "hostPort:containerPort/protocol"
    // Example: "8080:80/tcp"
    
    // First split by "/" to separate protocol
    parts := strings.Split(spec, "/")
    if len(parts) != 2 {
        return PortMapping{}, fmt.Errorf("invalid port mapping format: %s", spec)
    }
    
    portSpec := parts[0]
    protocol := parts[1]
    
    if protocol != "tcp" && protocol != "udp" {
        return PortMapping{}, fmt.Errorf("invalid protocol (must be tcp or udp): %s", protocol)
    }
    
    // Split port specification by ":"
    portParts := strings.Split(portSpec, ":")
    if len(portParts) != 2 {
        return PortMapping{}, fmt.Errorf("invalid port specification: %s", portSpec)
    }
    
    hostPort, err := strconv.Atoi(portParts[0])
    if err != nil {
        return PortMapping{}, fmt.Errorf("invalid host port: %s", portParts[0])
    }
    
    containerPort, err := strconv.Atoi(portParts[1])
    if err != nil {
        return PortMapping{}, fmt.Errorf("invalid container port: %s", portParts[1])
    }
    
    return PortMapping{
        HostPort:      hostPort,
        ContainerPort: containerPort,
        Protocol:      protocol,
    }, nil
}