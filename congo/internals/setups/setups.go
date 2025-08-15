package setups

import (
    "context"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    "syscall"
    
   // "congo/congo/internals/types"
	"congo/congo/internals/utils/userUtils"
	"congo/congo/internals/utils/generalUtils"
)



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
            uid, gid, err = userUtils.LookupUser(user)
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
    if err := userUtils.ValidateUserPermissions(uid, gid); err != nil {
        return err
    }
    
    log.Printf("Switching to user: uid=%d, gid=%d", uid, gid)
    
    // Get supplementary groups for the user
    groups, err := userUtils.GetUserGroups(username, gid)
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
    homeDir := generalUtils.getHomeDirectory(uid, username)
    if err := os.Setenv("HOME", homeDir); err != nil {
        log.Printf("Warning: failed to set HOME environment variable: %v", err)
    }
    
    log.Printf("User switch completed: USER=%s, HOME=%s", username, homeDir)
    
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
