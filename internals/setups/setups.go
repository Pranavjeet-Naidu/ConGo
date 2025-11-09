//go:build linux
// +build linux

package setups

import (
    "context"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "golang.org/x/sys/unix"
    "path/filepath"
    "congo/internals/types"
    "congo/internals/capabilities"
    "congo/internals/utils"
    "congo/internals/filesystem"
    //"congo/internals/logging"
    "congo/internals/monitoring"
    "congo/internals/cgroups"
)

func SetupUser(user string) error {
    return SetupUserWithContext(context.Background(), user)
}

func SetupUserWithContext(ctx context.Context, user string) error {
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
            uid, gid, err = utils.LookupUser(user)
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
    if err := utils.ValidateUserPermissions(uid, gid); err != nil {
        return err
    }
    
    log.Printf("Switching to user: uid=%d, gid=%d", uid, gid)
    
    // Get supplementary groups for the user
    groups, err := utils.GetUserGroups(username, gid)
    if err != nil {
        log.Printf("Warning: failed to get supplementary groups: %v", err)
        groups = []int{gid} // Fallback to primary group only
    }
    
    // Set supplementary groups for better security
    if err := unix.Setgroups(groups); err != nil {
        return fmt.Errorf("failed to set supplementary groups: %v", err)
    }
    
    // Set group ID first (must be done before setting user ID)
    if err := unix.Setgid(gid); err != nil {
        return fmt.Errorf("failed to set gid %d: %v", gid, err)
    }
    
    // Set user ID
    if err := unix.Setuid(uid); err != nil {
        return fmt.Errorf("failed to set uid %d: %v", uid, err)
    }
    
    // Update environment variables to reflect the user change
    if err := os.Setenv("USER", username); err != nil {
        log.Printf("Warning: failed to set USER environment variable: %v", err)
    }
    
    // Set HOME directory using actual home directory from user lookup when available
    homeDir := utils.GetHomeDirectory(uid, username)
    if err := os.Setenv("HOME", homeDir); err != nil {
        log.Printf("Warning: failed to set HOME environment variable: %v", err)
    }
    
    log.Printf("User switch completed: USER=%s, HOME=%s", username, homeDir)
    
    return nil
}

func SetupLogging(config *types.Config) error {
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
    if err := unix.Dup2(int(stdoutFile.Fd()), int(os.Stdout.Fd())); err != nil {
        stdoutFile.Close()
        stderrFile.Close()
        return fmt.Errorf("failed to redirect stdout: %v", err)
    }
    
    // Redirect standard error
    if err := unix.Dup2(int(stderrFile.Fd()), int(os.Stderr.Fd())); err != nil {
        stdoutFile.Close()
        stderrFile.Close()
        return fmt.Errorf("failed to redirect stderr: %v", err)
    }
    
    // Log that logging has been set up successfully
    fmt.Printf("Logging initialized: stdout -> %s, stderr -> %s\n", stdoutPath, stderrPath)
    
    return nil
}

func SetupMounts(mounts []types.Mount) error {
    for _, mount := range mounts {
        if err := os.MkdirAll(mount.Destination, 0755); err != nil {
            return fmt.Errorf("failed to create mount point: %v", err)
        }

        flags := unix.MS_BIND
        if mount.ReadOnly {
            flags |= unix.MS_RDONLY
        }

        if err := unix.Mount(mount.Source, mount.Destination, "", uintptr(flags), ""); err != nil {
            return fmt.Errorf("failed to mount: %v", err)
        }
    }
    return nil
}

func SetupEnv(envVars map[string]string) error {
    for key, value := range envVars {
        if err := os.Setenv(key, value); err != nil {
            return fmt.Errorf("failed to set environment variable: %v", err)
        }
    }
    return nil
}

func SetupContainer(config *types.Config) error {
    defer utils.Cleanup(config)

    // Set hostname
    if err := unix.Sethostname([]byte("container")); err != nil {
        return fmt.Errorf("error setting hostname: %v", err)
    }

    // Setup root filesystem
    if config.UseLayers {
        if err := filesystem.SetupLayeredRootfs(config); err != nil {
            return fmt.Errorf("error setting up layered rootfs: %v", err)
        }
    } else {
        if err := filesystem.SetupRootfs(config.Rootfs); err != nil {
            return fmt.Errorf("error setting up rootfs: %v", err)
        }
    }
    
    // Add capability setup early in the process
    if err := capabilities.SetupCapabilities(config); err != nil {
        return fmt.Errorf("error setting up capabilities: %v", err)
    }

    // Setup bind mounts
    if err := SetupMounts(config.Mounts); err != nil {
        return fmt.Errorf("error performing bind mounts: %v", err)
    }

    // Setup cgroups
    if err := cgroups.SetupCgroups(config); err != nil {
        return fmt.Errorf("error setting up cgroups: %v", err)
    }

    // Setup user (new functionality)
    if config.User != "" {
        if err := SetupUser(config.User); err != nil {
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
        if err := SetupLogging(config); err != nil {
            return fmt.Errorf("error setting up logging: %v", err)
        }
    }
    
    // Start resource monitoring if enabled
    if config.MonitorConfig.Enabled {
        if err := monitoring.StartResourceMonitoring(config); err != nil {
            return fmt.Errorf("error starting resource monitoring: %v", err)
        }
    }

    return nil
}

// SetupCapabilities configures Linux capabilities for the container
func SetupCapabilities(config *types.Config) error {
    caps := config.Capabilities
    
    if len(caps) == 0 {
        // Drop all capabilities by default
        log.Println("Dropping all capabilities")
        if err := capabilities.ClearAllCapabilities(); err != nil {
            return fmt.Errorf("failed to clear all capabilities: %v", err)
        }
        return nil
    }

    // Keep only specified capabilities
    log.Printf("Setting up capabilities: %v", caps)
    
    // First drop all capabilities
    if err := capabilities.ClearAllCapabilities(); err != nil {
        return fmt.Errorf("failed to clear all capabilities: %v", err)
    }
    
    // Then add back the ones specified
    for _, cap := range caps {
        capValue, exists := types.CapMap[cap]
        if !exists {
            return fmt.Errorf("unknown capability: %s", cap)
        }
        
        if err := capabilities.AddCapability(capValue); err != nil {
            return fmt.Errorf("failed to add capability %s: %v", cap, err)
        }
        log.Printf("Added capability: %s", cap)
    }
    
    return nil
}
