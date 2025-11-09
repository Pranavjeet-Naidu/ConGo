package utils

import(
	"os"
	"os/user"
	"log"
	"fmt"
	"strconv"
	"strings"
)

func LookupUserFallback(username string) (int, int, error) {
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
func LookupUser(username string) (int, int, error) {
    u, err := user.Lookup(username)
    if err != nil {
        // Fallback to manual parsing if standard library fails
        log.Printf("Standard user lookup failed, falling back to manual parsing: %v", err)
        return LookupUserFallback(username)
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

// validateUserPermissions checks if the current process has permission to switch to the target user
func ValidateUserPermissions(targetUID, targetGID int) error {
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
func GetUserGroups(username string, gid int) ([]int, error) {
    // For now, return just the primary group
    // In a more complete implementation, this would parse /etc/group
    // or use a more sophisticated user lookup
    return []int{gid}, nil
}
