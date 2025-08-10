
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/exec"
    "os/user"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
    "unsafe"
    // "golang.org/x/sys/unix"
    "net"
    "time"
	"encoding/json"
)


// capget syscall wrapper
func capget(header *CapUserHeader, data *CapUserData) error {
    _, _, errno := syscall.Syscall(syscall.SYS_CAPGET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}

// capset syscall wrapper
func capset(header *CapUserHeader, data *CapUserData) error {
    _, _, errno := syscall.Syscall(syscall.SYS_CAPSET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
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
        LogConfig: LoggingConfig{
            EnableLogging: false,
            MaxLogSize:    10 * 1024 * 1024, // Default 10 MB
        },
        MonitorConfig: MonitoringConfig{
            Enabled:          false,
            Interval:         30,  // Default 30 seconds
            MonitorCpu:       true,
            MonitorMemory:    true,
            MonitorProcesses: true,
        },
        Interactive: false,
        Detached:    false,
        StateDir:    getStateDir(),
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
        case "--log-dir":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing log directory")
            }
            config.LogConfig.LogDir = args[currentIdx+1]
            config.LogConfig.EnableLogging = true
            currentIdx += 2
        case "--log-max-size":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing maximum log size")
            }
            maxSize, err := strconv.ParseInt(args[currentIdx+1], 10, 64)
            if err != nil {
                return nil, fmt.Errorf("invalid log max size: %v", err)
            }
            config.LogConfig.MaxLogSize = maxSize
            currentIdx += 2
        case "--enable-monitor":
            config.MonitorConfig.Enabled = true
            currentIdx++
        case "--monitor-interval":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing monitoring interval")
            }
            interval, err := strconv.Atoi(args[currentIdx+1])
            if err != nil {
                return nil, fmt.Errorf("invalid monitoring interval: %v", err)
            }
            config.MonitorConfig.Interval = interval
            currentIdx += 2
        case "--monitor-stats-file":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing stats file path")
            }
            config.MonitorConfig.StatsFile = args[currentIdx+1]
            currentIdx += 2  
        case "--monitor-cpu":
            config.MonitorConfig.MonitorCpu = true
            currentIdx++
        case "--monitor-memory":
            config.MonitorConfig.MonitorMemory = true
            currentIdx++
        case "--monitor-processes":
            config.MonitorConfig.MonitorProcesses = true
            currentIdx++
		case "--interactive", "-i":
			config.Interactive = true
			currentIdx++
		case "--detach", "-d":
			config.Detached = true
			currentIdx++
		case "--id":
			if currentIdx+1 >= cmdIndex {
				return nil, fmt.Errorf("missing container ID")
			}
			config.ContainerID = args[currentIdx+1]
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

    // Setup logging if enabled
    if config.LogConfig.EnableLogging {
        if err := setupLogging(config); err != nil {
            return fmt.Errorf("error setting up logging: %v", err)
        }
    }
    
    // Start resource monitoring if enabled
    if config.MonitorConfig.Enabled {
        if err := startResourceMonitoring(config); err != nil {
            return fmt.Errorf("error starting resource monitoring: %v", err)
        }
    }

    return nil
}

func setupUser(user string) error {
    return setupUserWithContext(context.Background(), user)
}

func setupUserWithContext(ctx context.Context, user string) error {
    // Check for context cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Parse user specification (can be username, uid, or uid:gid)
    var uid, gid int
    var err error
    var username string
    
    // If user is empty, no user switching is needed
    if user == "" {
        return nil
    }
    
    // Input validation and parsing
    if strings.Contains(user, ":") {
        // Format: uid:gid
        parts := strings.Split(user, ":")
        if len(parts) != 2 {
            return fmt.Errorf("invalid user format, expected uid:gid")
        }
        
        uid, err = strconv.Atoi(parts[0])
        if err != nil {
            return fmt.Errorf("invalid uid: %v", err)
        }
        
        gid, err = strconv.Atoi(parts[1])
        if err != nil {
            return fmt.Errorf("invalid gid: %v", err)
        }
        
        username = parts[0] // Use uid as username for environment
    } else {
        // Check if it's a numeric uid
        if uid, err = strconv.Atoi(user); err == nil {
            // Use same value for gid as uid (common container practice)
            gid = uid
            username = user
        } else {
            // Try to look up username using standard library
            uid, gid, err = lookupUserImproved(user)
            if err != nil {
                return fmt.Errorf("failed to lookup user %s: %v", user, err)
            }
            username = user
        }
    }
    
    // Bounds checking for uid/gid
    if uid < 0 || uid > 65535 || gid < 0 || gid > 65535 {
        return fmt.Errorf("uid/gid out of valid range (0-65535): uid=%d, gid=%d", uid, gid)
    }
    
    // Security check: validate user permissions
    if err := validateUserPermissions(uid, gid); err != nil {
        return err
    }
    
    log.Printf("Switching to user: uid=%d, gid=%d", uid, gid)
    
    // Get supplementary groups for the user
    groups, err := getUserGroups(username, gid)
    if err != nil {
        log.Printf("Warning: failed to get supplementary groups: %v", err)
        groups = []int{gid} // Fallback to primary group only
    }
    
    // Set supplementary groups for better security
    if err := syscall.Setgroups(groups); err != nil {
        return fmt.Errorf("failed to set supplementary groups: %v", err)
    }
    
    // Set group ID first (must be done before setting user ID)
    if err := syscall.Setgid(gid); err != nil {
        return fmt.Errorf("failed to set gid %d: %v", gid, err)
    }
    
    // Set user ID
    if err := syscall.Setuid(uid); err != nil {
        return fmt.Errorf("failed to set uid %d: %v", uid, err)
    }
    
    // Update environment variables to reflect the user change
    if err := os.Setenv("USER", username); err != nil {
        log.Printf("Warning: failed to set USER environment variable: %v", err)
    }
    
    // Set HOME directory using actual home directory from user lookup when available
    homeDir := getHomeDirectory(uid, username)
    if err := os.Setenv("HOME", homeDir); err != nil {
        log.Printf("Warning: failed to set HOME environment variable: %v", err)
    }
    
    log.Printf("User switch completed: USER=%s, HOME=%s", username, homeDir)
    
    return nil
}

// lookupUserImproved uses the standard library for better user lookup
func lookupUserImproved(username string) (int, int, error) {
    u, err := user.Lookup(username)
    if err != nil {
        // Fallback to manual parsing if standard library fails
        log.Printf("Standard user lookup failed, falling back to manual parsing: %v", err)
        return lookupUserFallback(username)
    }
    
    uid, err := strconv.Atoi(u.Uid)
    if err != nil {
        return 0, 0, fmt.Errorf("invalid uid in user entry: %v", err)
    }
    
    gid, err := strconv.Atoi(u.Gid)
    if err != nil {
        return 0, 0, fmt.Errorf("invalid gid in user entry: %v", err)
    }
    
    return uid, gid, nil
}

// getHomeDirectory determines the appropriate home directory
func getHomeDirectory(uid int, username string) string {
    // Default for root
    if uid == 0 {
        return "/root"
    }
    
    // Try to get actual home directory from user lookup
    if u, err := user.LookupId(strconv.Itoa(uid)); err == nil && u.HomeDir != "" {
        return u.HomeDir
    }
    
    // Fallback to conventional path
    return fmt.Sprintf("/home/%s", username)
}

// lookupUserFallback provides manual /etc/passwd parsing as fallback
func lookupUserFallback(username string) (int, int, error) {
    passwdFile := "/etc/passwd"
    
    // Check if passwd file exists
    if _, err := os.Stat(passwdFile); os.IsNotExist(err) {
        return 0, 0, fmt.Errorf("user lookup not available (no /etc/passwd)")
    }
    
    // Read and parse /etc/passwd
    content, err := os.ReadFile(passwdFile)
    if err != nil {
        return 0, 0, fmt.Errorf("failed to read /etc/passwd: %v", err)
    }
    
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        fields := strings.Split(line, ":")
        if len(fields) >= 4 && fields[0] == username {
            uid, err := strconv.Atoi(fields[2])
            if err != nil {
                return 0, 0, fmt.Errorf("invalid uid in passwd entry: %v", err)
            }
            
            gid, err := strconv.Atoi(fields[3])
            if err != nil {
                return 0, 0, fmt.Errorf("invalid gid in passwd entry: %v", err)
            }
            
            return uid, gid, nil
        }
    }
    
    return 0, 0, fmt.Errorf("user %s not found", username)
}

// validateUserPermissions checks if the current process has permission to switch to the target user
func validateUserPermissions(targetUID, targetGID int) error {
    currentUID := os.Getuid()
    currentGID := os.Getgid()
    
    // Root can switch to any user
    if currentUID == 0 {
        return nil
    }
    
    // Non-root can only switch to same user/group
    if targetUID != currentUID || targetGID != currentGID {
        return fmt.Errorf("insufficient privileges: current user %d:%d cannot switch to %d:%d", 
            currentUID, currentGID, targetUID, targetGID)
    }
    
    return nil
}

// getUserGroups gets supplementary groups for a user
func getUserGroups(username string, gid int) ([]int, error) {
    // For now, return just the primary group
    // In a more complete implementation, this would parse /etc/group
    // or use a more sophisticated user lookup
    return []int{gid}, nil
}



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


func setupLogging(config *Config) error {
    if !config.LogConfig.EnableLogging {
        return nil
    }
    
    // Create log directory if it doesn't exist
    if err := os.MkdirAll(config.LogConfig.LogDir, 0755); err != nil {
        return fmt.Errorf("failed to create log directory: %v", err)
    }
    
    // Create stdout log file
    stdoutPath := filepath.Join(config.LogConfig.LogDir, fmt.Sprintf("container-%d-stdout.log", os.Getpid()))
    stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return fmt.Errorf("failed to open stdout log file: %v", err)
    }
    
    // Create stderr log file
    stderrPath := filepath.Join(config.LogConfig.LogDir, fmt.Sprintf("container-%d-stderr.log", os.Getpid()))
    stderrFile, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        stdoutFile.Close()
        return fmt.Errorf("failed to open stderr log file: %v", err)
    }
    
    // Redirect standard output
    if err := syscall.Dup2(int(stdoutFile.Fd()), int(os.Stdout.Fd())); err != nil {
        stdoutFile.Close()
        stderrFile.Close()
        return fmt.Errorf("failed to redirect stdout: %v", err)
    }
    
    // Redirect standard error
    if err := syscall.Dup2(int(stderrFile.Fd()), int(os.Stderr.Fd())); err != nil {
        stdoutFile.Close()
        stderrFile.Close()
        return fmt.Errorf("failed to redirect stderr: %v", err)
    }
    
    // Log that logging has been set up successfully
    fmt.Printf("Logging initialized: stdout -> %s, stderr -> %s\n", stdoutPath, stderrPath)
    
    return nil
}



func startResourceMonitoring(config *Config) error {
    if !config.MonitorConfig.Enabled {
        return nil
    }
    
    // Set default interval if not specified
    if config.MonitorConfig.Interval <= 0 {
        config.MonitorConfig.Interval = 30 // Default to 30 seconds
    }
    
    // Set default stats file if not specified
    if config.MonitorConfig.StatsFile == "" {
        if config.LogConfig.EnableLogging {
            config.MonitorConfig.StatsFile = filepath.Join(config.LogConfig.LogDir, 
                fmt.Sprintf("container-%d-stats.log", os.Getpid()))
        } else {
            return fmt.Errorf("stats file must be specified when logging is disabled")
        }
    }
    
    // Enable all metrics by default if none specified
    if !config.MonitorConfig.MonitorCpu && 
       !config.MonitorConfig.MonitorMemory && 
       !config.MonitorConfig.MonitorProcesses {
        config.MonitorConfig.MonitorCpu = true
        config.MonitorConfig.MonitorMemory = true
        config.MonitorConfig.MonitorProcesses = true
    }
    
    // Create stats file
    statsFile, err := os.OpenFile(
        config.MonitorConfig.StatsFile,
        os.O_CREATE|os.O_WRONLY|os.O_APPEND,
        0644,
    )
    if err != nil {
        return fmt.Errorf("failed to open stats file: %v", err)
    }
    
    // Start monitoring in a separate goroutine
    go func() {
        ticker := time.NewTicker(time.Duration(config.MonitorConfig.Interval) * time.Second)
        defer ticker.Stop()
        defer statsFile.Close()
        
        fmt.Fprintf(statsFile, "=== Resource monitoring started at %s ===\n", 
            time.Now().Format(time.RFC3339))
            
        for {
            select {
            case <-ticker.C:
                stats, err := collectResourceStats(config)
                if err != nil {
                    fmt.Fprintf(statsFile, "Error collecting stats: %v\n", err)
                    continue
                }
                
                // Write stats to file
                timestamp := time.Now().Format(time.RFC3339)
                fmt.Fprintf(statsFile, "[%s] %s\n", timestamp, stats)
            }
        }
    }()
    
    fmt.Printf("Resource monitoring started: stats file -> %s, interval -> %ds\n", 
        config.MonitorConfig.StatsFile, config.MonitorConfig.Interval)
    
    return nil
}

func collectResourceStats(config *Config) (string, error) {
    containerID := fmt.Sprintf("container-%d", os.Getpid())
    var stats strings.Builder
    
    // Collect CPU stats
    if config.MonitorConfig.MonitorCpu {
        // For cgroup v2
        cpuStatPath := filepath.Join("/sys/fs/cgroup/cpu.stat")
        if _, err := os.Stat(cpuStatPath); err == nil {
            cpuData, err := os.ReadFile(cpuStatPath)
            if err == nil {
                stats.WriteString("CPU: ")
                stats.WriteString(strings.Replace(string(cpuData), "\n", " ", -1))
                stats.WriteString(" | ")
            }
        } else {
            // Fallback to cgroup v1
            cpuStatPath := filepath.Join("/sys/fs/cgroup/cpu", containerID, "cpu.stat")
            cpuData, err := os.ReadFile(cpuStatPath)
            if err == nil {
                stats.WriteString("CPU: ")
                stats.WriteString(strings.Replace(string(cpuData), "\n", " ", -1))
                stats.WriteString(" | ")
            }
        }
    }
    
    // Collect memory stats
    if config.MonitorConfig.MonitorMemory {
        // For cgroup v2
        memStatPath := filepath.Join("/sys/fs/cgroup/memory.current")
        if _, err := os.Stat(memStatPath); err == nil {
            memData, err := os.ReadFile(memStatPath)
            if err == nil {
                memBytes, _ := strconv.ParseInt(strings.TrimSpace(string(memData)), 10, 64)
                memMB := float64(memBytes) / 1024 / 1024
                stats.WriteString(fmt.Sprintf("Memory: %.2f MB | ", memMB))
            }
        } else {
            // Fallback to cgroup v1
            memStatPath := filepath.Join("/sys/fs/cgroup/memory", containerID, "memory.usage_in_bytes")
            memData, err := os.ReadFile(memStatPath)
            if err == nil {
                memBytes, _ := strconv.ParseInt(strings.TrimSpace(string(memData)), 10, 64)
                memMB := float64(memBytes) / 1024 / 1024
                stats.WriteString(fmt.Sprintf("Memory: %.2f MB | ", memMB))
            }
        }
    }
    
    // Collect process count
    if config.MonitorConfig.MonitorProcesses {
        // For cgroup v2
        pidsStatPath := filepath.Join("/sys/fs/cgroup/pids.current")
        if _, err := os.Stat(pidsStatPath); err == nil {
            pidsData, err := os.ReadFile(pidsStatPath)
            if err == nil {
                stats.WriteString(fmt.Sprintf("Processes: %s", strings.TrimSpace(string(pidsData))))
            }
        } else {
            // Fallback to cgroup v1
            pidsStatPath := filepath.Join("/sys/fs/cgroup/pids", containerID, "pids.current")
            pidsData, err := os.ReadFile(pidsStatPath)
            if err == nil {
                stats.WriteString(fmt.Sprintf("Processes: %s", strings.TrimSpace(string(pidsData))))
            }
        }
    }
    
    return stats.String(), nil
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


func removeContainer(containerID string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check if container is running
    if state.Status == "running" {
        return fmt.Errorf("cannot remove running container %s, stop it first", containerID)
    }
    
    // Remove container state file
    stateFile := filepath.Join(getStateDir(), containerID+".json")
    if err := os.Remove(stateFile); err != nil {
        return fmt.Errorf("failed to remove container state file: %v", err)
    }
    
    // Clean up log directory if exists
    if state.LogDir != "" {
        if err := os.RemoveAll(state.LogDir); err != nil {
            log.Printf("Warning: failed to remove log directory: %v", err)
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
            // for now unnecessary , maybe for later :D
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

func viewContainerLogs(containerID string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Determine log location - use log directory from state if available
    var logDir string
    if state.LogDir != "" {
        logDir = state.LogDir
    } else {
        logDir = filepath.Join("/var/log/congo", containerID)
    }
    stdoutLog := filepath.Join(logDir, "stdout.log")
    stderrLog := filepath.Join(logDir, "stderr.log")
    
    // Check if logs exist
    if _, err := os.Stat(stdoutLog); os.IsNotExist(err) {
        return fmt.Errorf("no logs found for container %s", containerID)
    }
    
    // Print stdout logs
    fmt.Println("=== STDOUT ===")
    stdout, err := os.ReadFile(stdoutLog)
    if err != nil {
        return fmt.Errorf("failed to read stdout log: %v", err)
    }
    fmt.Println(string(stdout))
    
    // Print stderr logs
    fmt.Println("=== STDERR ===")
    stderr, err := os.ReadFile(stderrLog)
    if err != nil {
        return fmt.Errorf("failed to read stderr log: %v", err)
    }
    fmt.Println(string(stderr))
    
    return nil
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

    // Handle container lifecycle commands
    switch os.Args[1] {
    case "create":
        // Create a new container but don't start it
        config, err := parseConfig(os.Args[2:], false)
        if err != nil {
            log.Fatalf("Error parsing config: %v", err)
        }
        
        // Generate a unique container ID if not provided
        if config.ContainerID == "" {
            config.ContainerID = fmt.Sprintf("congo-%d", time.Now().UnixNano())
        }
        
        // Initialize container state
        config.State = ContainerState{
            ID:        config.ContainerID,
            Status:    "created",
            CreatedAt: time.Now(),
            Command:   config.Command,
            RootDir:   config.Rootfs,
        }
        
        // Save the container state
        if err := saveContainerState(config.ContainerID, config.State); err != nil {
            log.Fatalf("Error saving container state: %v", err)
        }
        
        fmt.Printf("Container created: %s\n", config.ContainerID)

	case "commit":
		// Create an image from a container
		if len(os.Args) < 4 {
			log.Fatalf("Usage: %s commit <container-id> <image-name>", os.Args[0])
		}
		containerID := os.Args[2]
		imageName := os.Args[3]
		
		if err := commitContainer(containerID, imageName); err != nil {
			log.Fatalf("Error committing container: %v", err)
		}
		
		fmt.Printf("Container %s committed to image: %s\n", containerID, imageName)
	
	case "logs":
		// View container logs
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s logs <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := viewContainerLogs(containerID); err != nil {
			log.Fatalf("Error viewing container logs: %v", err)
		}
	
        
    case "start":
        // Start an existing container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s start <container-id>", os.Args[0])
        }
        containerID := os.Args[2]
        
        if err := startContainer(containerID, os.Args[3:]); err != nil {
            log.Fatalf("Error starting container: %v", err)
        }
        
        fmt.Printf("Container started: %s\n", containerID)
    
	case "rm":
		// Remove a container
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s rm <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := removeContainer(containerID); err != nil {
			log.Fatalf("Error removing container: %v", err)
		}
		
		fmt.Printf("Container removed: %s\n", containerID)
	
    case "stop":
        // Stop a running container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s stop <container-id> [--force]", os.Args[0])
        }
        containerID := os.Args[2]
        force := len(os.Args) > 3 && os.Args[3] == "--force"
        
        if err := stopContainer(containerID, force); err != nil {
            log.Fatalf("Error stopping container: %v", err)
        }
        
        fmt.Printf("Container stopped: %s\n", containerID)
        
    case "restart":
        // Restart a container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s restart <container-id>", os.Args[0])
        }
        containerID := os.Args[2]
        
        if err := restartContainer(containerID); err != nil {
            log.Fatalf("Error restarting container: %v", err)
        }
        
        fmt.Printf("Container restarted: %s\n", containerID)
        
    case "exec":
        // Execute a command in a running container
        if len(os.Args) < 4 {
            log.Fatalf("Usage: %s exec <container-id> <command> [args...]", os.Args[0])
        }
        containerID := os.Args[2]
        command := os.Args[3:]
        
        if err := execInContainer(containerID, command); err != nil {
            log.Fatalf("Error executing command in container: %v", err)
        }
        
    case "shell":
        // Start an interactive shell in a container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s shell <container-id>", os.Args[0])
        }
        containerID := os.Args[2]
        
        // Default to bash, but fall back to sh if not available
        shell := []string{"/bin/bash"}
        if err := execInContainer(containerID, shell); err != nil {
            // Try sh if bash fails
            if err := execInContainer(containerID, []string{"/bin/sh"}); err != nil {
                log.Fatalf("Error starting shell in container: %v", err)
            }
        }
        
    case "ps":
        // List containers
        containers, err := listContainers()
        if err != nil {
            log.Fatalf("Error listing containers: %v", err)
        }
        
        // Print container information
        fmt.Printf("%-20s %-10s %-20s %-30s\n", "CONTAINER ID", "STATUS", "CREATED", "COMMAND")
        for _, container := range containers {
            cmdStr := strings.Join(container.Command, " ")
            if len(cmdStr) > 30 {
                cmdStr = cmdStr[:27] + "..."
            }
            fmt.Printf("%-20s %-10s %-20s %-30s\n", 
                container.ID, 
                container.Status, 
                container.CreatedAt.Format(time.RFC3339), 
                cmdStr)
        }
        
    case "child":
        // Handle child process (container process)
        isChild := true
        config, err := parseConfig(os.Args[2:], isChild)
        if err != nil {
            log.Fatalf("Error parsing config: %v", err)
        }

        if err := validateConfig(config); err != nil {
            log.Fatalf("Invalid config: %v", err)
        }

        if err := setupContainer(config); err != nil {
            log.Fatalf("Error setting up container: %v", err)
        }

        // Check if interactive mode is requested
        if config.Interactive {
            // In interactive mode, start a shell
            if err := syscall.Exec("/bin/bash", []string{"bash"}, os.Environ()); err != nil {
                // Try to fall back to sh if bash is not available
                if err := syscall.Exec("/bin/sh", []string{"sh"}, os.Environ()); err != nil {
                    log.Fatalf("Error executing shell: %v", err)
                }
            }
        } else {
            // In non-interactive mode, execute the specified command
            if err := syscall.Exec(config.Command[0], config.Command, os.Environ()); err != nil {
                log.Fatalf("Error executing command: %v", err)
            }
        }

	case "pause":
		// Pause a running container
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s pause <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := pauseContainer(containerID); err != nil {
			log.Fatalf("Error pausing container: %v", err)
		}
		
		fmt.Printf("Container paused: %s\n", containerID)
		
	case "unpause":
		// Unpause a paused container
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s unpause <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := unpauseContainer(containerID); err != nil {
			log.Fatalf("Error unpausing container: %v", err)
		}
		
		fmt.Printf("Container unpaused: %s\n", containerID)
	case "update":
		// Update container resource limits
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s update <container-id> [--memory=<limit>] [--cpu=<shares>] [--pids=<limit>]", os.Args[0])
		}
		containerID := os.Args[2]
		
		var memory, cpu string
		var pids int
		
		// Parse update options
		for i := 3; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "--memory=") {
				memory = strings.TrimPrefix(os.Args[i], "--memory=")
			} else if strings.HasPrefix(os.Args[i], "--cpu=") {
				cpu = strings.TrimPrefix(os.Args[i], "--cpu=")
			} else if strings.HasPrefix(os.Args[i], "--pids=") {
				pidLimit := strings.TrimPrefix(os.Args[i], "--pids=")
				var err error
				pids, err = strconv.Atoi(pidLimit)
				if err != nil {
					log.Fatalf("Invalid pid limit: %v", err)
				}
			}
		}
		
		if err := updateContainerResources(containerID, memory, cpu, pids); err != nil {
			log.Fatalf("Error updating container resources: %v", err)
		}
		
		fmt.Printf("Container %s resources updated\n", containerID)
	
    case "volume-add":
		// Add a volume to a running container
		if len(os.Args) < 5 {
			log.Fatalf("Usage: %s volume-add <container-id> <host-path> <container-path> [ro]", os.Args[0])
		}
		containerID := os.Args[2]
		hostPath := os.Args[3]
		containerPath := os.Args[4]
		readOnly := len(os.Args) > 5 && os.Args[5] == "ro"
		
		if err := addVolumeToContainer(containerID, hostPath, containerPath, readOnly); err != nil {
			log.Fatalf("Error adding volume: %v", err)
		}
		
		fmt.Printf("Volume added to container %s: %s -> %s\n", containerID, hostPath, containerPath)
		
	case "volume-remove":
		// Remove a volume from a running container
		if len(os.Args) < 4 {
			log.Fatalf("Usage: %s volume-remove <container-id> <container-path>", os.Args[0])
		}
		containerID := os.Args[2]
		containerPath := os.Args[3]
		
		if err := removeVolumeFromContainer(containerID, containerPath); err != nil {
			log.Fatalf("Error removing volume: %v", err)
		}
		
		fmt.Printf("Volume removed from container %s: %s\n", containerID, containerPath)
	 
    case "run":
		
        // Create and start a container in one step
        config, err := parseConfig(os.Args[2:], false)
        if err != nil {
            log.Fatalf("Error parsing config: %v", err)
        }
        config.State.Network.ContainerIP = config.Network.ContainerIP
		config.State.Network.Bridge = config.Network.Bridge
		config.State.Network.PortMaps = config.Network.PortMaps
        if err := validateConfig(config); err != nil {
            log.Fatalf("Invalid config: %v", err)
        }
        
        // Generate a unique container ID
        if config.ContainerID == "" {
            config.ContainerID = fmt.Sprintf("congo-%d", time.Now().UnixNano())
        }
        
        // Initialize container state
        config.State = ContainerState{
            ID:        config.ContainerID,
            Status:    "created", // Will be updated to "running" when started
            CreatedAt: time.Now(),
            Command:   config.Command,
            RootDir:   config.Rootfs,
        }
        
        // Save the container state
        if err := saveContainerState(config.ContainerID, config.State); err != nil {
            log.Fatalf("Error saving container state: %v", err)
        }
        
        // Start the container
        cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        cmd.SysProcAttr = &syscall.SysProcAttr{
            Cloneflags: syscall.CLONE_NEWUTS |
                syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS |
                syscall.CLONE_NEWNET |
                syscall.CLONE_NEWIPC |
                syscall.CLONE_NEWUSER,
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
        
        if err := cmd.Start(); err != nil {
            log.Fatalf("Error starting container: %v", err)
        }
        
        // Update container state
        config.State.Status = "running"
        config.State.Pid = cmd.Process.Pid
        if err := saveContainerState(config.ContainerID, config.State); err != nil {
            log.Printf("Warning: failed to update container state: %v", err)
        }
        
        fmt.Printf("Container started: %s\n", config.ContainerID)
        
        // If not detached, wait for the container to exit
        if !config.Detached {
            if err := cmd.Wait(); err != nil {
                log.Printf("Container exited with error: %v", err)
            }
            
            // Update container state to stopped
            config.State.Status = "stopped"
            config.State.Pid = 0
            if err := saveContainerState(config.ContainerID, config.State); err != nil {
                log.Printf("Warning: failed to update container state: %v", err)
            }
        }
        
    default:
        log.Fatalf("Unknown command: %s", os.Args[1])
    }
}

func updateContainerResources(containerID, memory, cpu string, pids int) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check if container exists
    if state.Status == "" {
        return fmt.Errorf("container %s does not exist", containerID)
    }
    
    // Update memory limit if specified
    if memory != "" {
        memoryPath := filepath.Join("/sys/fs/cgroup/memory", "congo-"+containerID, "memory.limit_in_bytes")
        if err := os.WriteFile(memoryPath, []byte(memory), 0644); err != nil {
            return fmt.Errorf("failed to update memory limit: %v", err)
        }
        state.ResourceLimits.Memory = memory
    }
    
    // Update CPU shares if specified
    if cpu != "" {
        cpuPath := filepath.Join("/sys/fs/cgroup/cpu", "congo-"+containerID, "cpu.shares")
        if err := os.WriteFile(cpuPath, []byte(cpu), 0644); err != nil {
            return fmt.Errorf("failed to update CPU shares: %v", err)
        }
        state.ResourceLimits.CPU = cpu
    }
    
    // Update process limit if specified
    if pids > 0 {
        pidsPath := filepath.Join("/sys/fs/cgroup/pids", "congo-"+containerID, "pids.max")
        if err := os.WriteFile(pidsPath, []byte(strconv.Itoa(pids)), 0644); err != nil {
            return fmt.Errorf("failed to update process limit: %v", err)
        }
        state.ResourceLimits.ProcessLimit = pids
    }
    
    // Save updated state
    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to save container state: %v", err)
    }
    
    return nil
}

func commitContainer(containerID, imageName string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check container status
    if state.Status == "running" {
        log.Printf("Warning: Committing a running container may result in inconsistent image")
    }
    
    // Create image directory
    imageDir := filepath.Join("/var/lib/congo/images", imageName)
    if err := os.MkdirAll(imageDir, 0755); err != nil {
        return fmt.Errorf("failed to create image directory: %v", err)
    }
    
    // Create a tarball of the container filesystem
    tarPath := filepath.Join(imageDir, "rootfs.tar")
    tarCmd := exec.Command("tar", "-C", state.RootDir, "-cf", tarPath, ".")
    if err := tarCmd.Run(); err != nil {
        return fmt.Errorf("failed to create image tarball: %v", err)
    }
    
    // Create image metadata file
    metadata := struct {
        ImageName   string            `json:"name"`
        CreatedAt   time.Time         `json:"created_at"`
        ContainerID string            `json:"container_id"`
        EnvVars     map[string]string `json:"env_vars"`
        Command     []string          `json:"command"`
    }{
        ImageName:   imageName,
        CreatedAt:   time.Now(),
        ContainerID: containerID,
        EnvVars:     state.EnvVars,
        Command:     state.Command,
    }
    
    metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal image metadata: %v", err)
    }
    
    metadataPath := filepath.Join(imageDir, "metadata.json")
    if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
        return fmt.Errorf("failed to write image metadata: %v", err)
    }
    
    return nil
}


func pauseContainer(containerID string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check if container is running
    if state.Status != "running" {
        return fmt.Errorf("container %s is not in running state", containerID)
    }
    
    // Create freezer cgroup directory if it doesn't exist
    freezerDir := filepath.Join("/sys/fs/cgroup/freezer", "congo-"+containerID)
    if err := os.MkdirAll(freezerDir, 0755); err != nil {
        return fmt.Errorf("failed to create freezer cgroup: %v", err)
    }
    
    // Add container process to the freezer cgroup
    if err := os.WriteFile(filepath.Join(freezerDir, "cgroup.procs"), 
        []byte(strconv.Itoa(state.Pid)), 0644); err != nil {
        return fmt.Errorf("failed to add process to freezer cgroup: %v", err)
    }
    
    // Freeze the container
    if err := os.WriteFile(filepath.Join(freezerDir, "freezer.state"), 
        []byte("FROZEN"), 0644); err != nil {
        return fmt.Errorf("failed to freeze container: %v", err)
    }
    
    // Update container state
    state.Status = "paused"
    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to update container state: %v", err)
    }
    
    return nil
}

func unpauseContainer(containerID string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check if container is paused
    if state.Status != "paused" {
        return fmt.Errorf("container %s is not in paused state", containerID)
    }
    
    // Path to container's freezer cgroup
    freezerDir := filepath.Join("/sys/fs/cgroup/freezer", "congo-"+containerID)
    
    // Unfreeze the container
    if err := os.WriteFile(filepath.Join(freezerDir, "freezer.state"), 
        []byte("THAWED"), 0644); err != nil {
        return fmt.Errorf("failed to unfreeze container: %v", err)
    }
    
    // Update container state
    state.Status = "running"
    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to update container state: %v", err)
    }
    
    return nil
}

func addVolumeToContainer(containerID, hostPath, containerPath string, readOnly bool) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check if container is running
    if state.Status != "running" {
        return fmt.Errorf("container %s is not running", containerID)
    }
    
    // Create the mount point inside the container
    createDirCmd := fmt.Sprintf("mkdir -p %s", containerPath)
    if err := execInContainer(containerID, []string{"/bin/sh", "-c", createDirCmd}); err != nil {
        return fmt.Errorf("failed to create mount point: %v", err)
    }
    
    // Prepare nsenter command to enter container mount namespace
    args := []string{
        "-t", strconv.Itoa(state.Pid),
        "-m",
        "--",
    }
    
    // Use mount to attach the volume
    mountOpts := ""
    if readOnly {
        mountOpts = "-o ro"
    }
    
    mountCmd := fmt.Sprintf("mount %s --bind %s %s", mountOpts, hostPath, containerPath)
    args = append(args, "/bin/sh", "-c", mountCmd)
    
    cmd := exec.Command("nsenter", args...)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to mount volume: %v", err)
    }
    
    // Update container state with the new mount
    newMount := Mount{
        Source:      hostPath,
        Destination: containerPath,
        ReadOnly:    readOnly,
    }
    
    state.Mounts = append(state.Mounts, newMount)
    
    // Save updated container state
    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to update container state: %v", err)
    }
    
    return nil
}

func removeVolumeFromContainer(containerID, containerPath string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }
    
    // Check if container is running
    if state.Status != "running" {
        return fmt.Errorf("container %s is not running", containerID)
    }
    
    // Check if the mount exists
    mountExists := false
    mountIndex := -1
    for i, mount := range state.Mounts {
        if mount.Destination == containerPath {
            mountExists = true
            mountIndex = i
            break
        }
    }
    
    if !mountExists {
        return fmt.Errorf("no mount found at path %s", containerPath)
    }
    
    // Prepare nsenter command to enter container mount namespace
    args := []string{
        "-t", strconv.Itoa(state.Pid),
        "-m",
        "--",
        "umount",
        containerPath,
    }
    
    cmd := exec.Command("nsenter", args...)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to unmount volume: %v", err)
    }
    
    // Update container state by removing the mount
    state.Mounts = append(state.Mounts[:mountIndex], state.Mounts[mountIndex+1:]...)
    
    // Save updated container state
    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to update container state: %v", err)
    }
    
    return nil
}

func startContainer(containerID string, args []string) error {
    // Check if container exists
    stateFile := filepath.Join(getStateDir(), containerID+".json")
    if _, err := os.Stat(stateFile); os.IsNotExist(err) {
        return fmt.Errorf("container %s does not exist", containerID)
    }

    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }

    // Check if container is already running
    if state.Status == "running" {
        return fmt.Errorf("container %s is already running", containerID)
    }

    // Reconstruct container configuration
    containerArgs := buildArgsFromState(state)
    
    // Merge with any additional args provided
    if len(args) > 0 {
        // Override command if specified
        containerArgs = append(containerArgs[:len(containerArgs)-len(state.Command)], args...)
    }
    
    // Build command to start the container
    cmd := exec.Command("/proc/self/exe", append([]string{"child"}, containerArgs...)...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Cloneflags: syscall.CLONE_NEWUTS |
            syscall.CLONE_NEWPID |
            syscall.CLONE_NEWNS |
            syscall.CLONE_NEWNET |
            syscall.CLONE_NEWIPC |
            syscall.CLONE_NEWUSER,
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

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start container: %v", err)
    }

    // Update container state
    state.Pid = cmd.Process.Pid
    state.Status = "running"
    state.CreatedAt = time.Now()

    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to save container state: %v", err)
    }

    if !state.Detached {
        if err := cmd.Wait(); err != nil {
            return fmt.Errorf("container process exited with error: %v", err)
        }
        
        // Update state after container exits
        state.Status = "stopped"
        state.Pid = 0
        if err := saveContainerState(containerID, state); err != nil {
            return fmt.Errorf("failed to update container state: %v", err)
        }
    }

    return nil
}

func stopContainer(containerID string, force bool) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }

    // Check if container is running
    if state.Status != "running" {
        return fmt.Errorf("container %s is not running", containerID)
    }

    // Send signal to container process
    process, err := os.FindProcess(state.Pid)
    if err != nil {
        return fmt.Errorf("failed to find container process: %v", err)
    }

    // Send SIGTERM or SIGKILL
    signal := syscall.SIGTERM
    if force {
        signal = syscall.SIGKILL
    }

    if err := process.Signal(signal); err != nil {
        return fmt.Errorf("failed to send signal to container: %v", err)
    }

    // Wait for the process to exit (with timeout)
    done := make(chan error)
    go func() {
        _, err := process.Wait()
        done <- err
    }()

    // Wait for process to exit or timeout
    var waitErr error
    select {
    case waitErr = <-done:
        // Process exited
    case <-time.After(10 * time.Second):
        if !force {
            // Timeout, try SIGKILL
            log.Printf("Container didn't exit after SIGTERM, sending SIGKILL")
            if err := process.Kill(); err != nil {
                return fmt.Errorf("failed to forcefully kill container: %v", err)
            }
            waitErr = <-done
        } else {
            return fmt.Errorf("container failed to exit even after SIGKILL")
        }
    }

    if waitErr != nil {
        log.Printf("Container exited with error: %v", waitErr)
    }

    // Clean up network resources
    if err := cleanupContainerNetwork(state.Pid); err != nil {
        log.Printf("Warning: failed to clean up container network: %v", err)
    }

    // Clean up any port forwarding rules
    if err := cleanupPortForwarding(containerID); err != nil {
        log.Printf("Warning: failed to clean up port forwarding rules: %v", err)
    }

    // Update container state
    state.Status = "stopped"
    state.Pid = 0
    if err := saveContainerState(containerID, state); err != nil {
        return fmt.Errorf("failed to update container state: %v", err)
    }

    return nil
}

func cleanupContainerNetwork(pid int) error {
    // Clean up veth pair - the host side only, container side vanishes with namespace
    hostVeth := fmt.Sprintf("hveth%d", pid)
    
    // Check if the interface exists before trying to remove it
    if _, err := net.InterfaceByName(hostVeth); err == nil {
        // Remove host veth interface
        if err := exec.Command("ip", "link", "del", hostVeth).Run(); err != nil {
            return fmt.Errorf("failed to remove host veth interface: %v", err)
        }
    }
    
    return nil
}

func cleanupPortForwarding(containerID string) error {
    // Get container state to find any port mappings
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state for port cleanup: %v", err)
    }
    
    
    // Check for network configuration in the state
    if state.Network.ContainerIP == "" || len(state.Network.PortMaps) == 0 {
        // No port mappings to clean up
        return nil
    }
    
    // Remove iptables rules for each port mapping
    for _, port := range state.Network.PortMaps {
        rule := fmt.Sprintf(
            "-t nat -D PREROUTING -p %s --dport %d -j DNAT --to-destination %s:%d",
            port.Protocol,
            port.HostPort,
            state.Network.ContainerIP,
            port.ContainerPort,
        )
        
        if err := exec.Command("iptables", strings.Split(rule, " ")...).Run(); err != nil {
            log.Printf("Warning: failed to remove port forwarding rule: %v", err)
        }
        
        // Also clean up any related MASQUERADE rules
        masqRule := fmt.Sprintf(
            "-t nat -D POSTROUTING -p %s -s %s --dport %d -j MASQUERADE",
            port.Protocol,
            state.Network.ContainerIP,
            port.ContainerPort,
        )
        
        if err := exec.Command("iptables", strings.Split(masqRule, " ")...).Run(); err != nil {
            log.Printf("Warning: failed to remove masquerade rule: %v", err)
        }
    }
    
    log.Printf("Cleaned up port forwarding rules for container %s", containerID)
    return nil
}

func restartContainer(containerID string) error {
    // First stop the container
    if err := stopContainer(containerID, false); err != nil {
        return fmt.Errorf("failed to stop container for restart: %v", err)
    }

    // Load container state to get command arguments
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }

    // Reconstruct arguments needed to start the container
    args := buildArgsFromState(state)

    // Start the container again
    if err := startContainer(containerID, args); err != nil {
        return fmt.Errorf("failed to start container for restart: %v", err)
    }

    return nil
}

func execInContainer(containerID string, command []string) error {
    // Load container state
    state, err := loadContainerState(containerID)
    if err != nil {
        return fmt.Errorf("failed to load container state: %v", err)
    }

    // Check if container is running
    if state.Status != "running" {
        return fmt.Errorf("container %s is not running", containerID)
    }

    // Prepare nsenter command to enter container namespaces
    args := []string{
        "-t", strconv.Itoa(state.Pid),
        "-m", "-u", "-i", "-n", "-p",
    }

    // Add the command to execute
    args = append(args, "--")
    args = append(args, command...)

    // Execute command inside container namespaces
    cmd := exec.Command("nsenter", args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to execute command in container: %v", err)
    }

    return nil
}

func getStateDir() string {
    dir := "/var/run/congo"
    if err := os.MkdirAll(dir, 0755); err != nil {
        // Fallback to temp directory if can't create in /var/run
        dir = filepath.Join(os.TempDir(), "congo")
        os.MkdirAll(dir, 0755)
    }
    return dir
}

func saveContainerState(containerID string, state ContainerState) error {
    stateDir := getStateDir()
    stateFile := filepath.Join(stateDir, containerID+".json")
    
    data, err := json.Marshal(state)
    if err != nil {
        return fmt.Errorf("failed to marshal container state: %v", err)
    }
    
    if err := os.WriteFile(stateFile, data, 0644); err != nil {
        return fmt.Errorf("failed to write container state file: %v", err)
    }
    
    return nil
}

func loadContainerState(containerID string) (ContainerState, error) {
    stateDir := getStateDir()
    stateFile := filepath.Join(stateDir, containerID+".json")
    
    data, err := os.ReadFile(stateFile)
    if err != nil {
        return ContainerState{}, fmt.Errorf("failed to read container state file: %v", err)
    }
    
    var state ContainerState
    if err := json.Unmarshal(data, &state); err != nil {
        return ContainerState{}, fmt.Errorf("failed to unmarshal container state: %v", err)
    }
    
    return state, nil
}

func listContainers() ([]ContainerState, error) {
    stateDir := getStateDir()
    files, err := os.ReadDir(stateDir)
    if err != nil {
        if os.IsNotExist(err) {
            return []ContainerState{}, nil
        }
        return nil, fmt.Errorf("failed to read state directory: %v", err)
    }
    
    var containers []ContainerState
    for _, file := range files {
        if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
            containerID := strings.TrimSuffix(file.Name(), ".json")
            state, err := loadContainerState(containerID)
            if err != nil {
                log.Printf("Warning: failed to load state for container %s: %v", containerID, err)
                continue
            }
            containers = append(containers, state)
        }
    }
    
    return containers, nil
}

func buildArgsFromState(state ContainerState) []string {
    // Create a basic set of environment variables
    args := []string{
        "/bin/sh",                  // Default PATH
        "/root",                    // Default HOME
        "root",                     // Default USER
        "/bin/bash",                // Default SHELL
        "xterm",                    // Default TERM
        "en_US.UTF-8",              // Default LANG
    }
    
    // Add container rootfs option
    args = append(args, "--rootfs", state.RootDir)
    
    // Add container ID
    args = append(args, "--id", state.ID)
    
    // Add command separator
    args = append(args, "--")
    
    // Add the command from the state
    args = append(args, state.Command...)
    
    return args
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

