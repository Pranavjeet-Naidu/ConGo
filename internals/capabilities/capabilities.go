//go:build linux
// +build linux

package capabilities

import (
    "fmt"
    "log"
    "unsafe"
    "golang.org/x/sys/unix"
    "congo/internals/types"
)

// capget unix wrapper
func capget(header *types.CapUserHeader, data *types.CapUserData) error {
    _, _, errno := unix.Syscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}

// capset unix wrapper
func capset(header *types.CapUserHeader, data *types.CapUserData) error {
    _, _, errno := unix.Syscall(unix.SYS_CAPSET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}

// SetupCapabilities configures Linux capabilities for the container
// Fixed signature to accept *types.Config instead of []string
func SetupCapabilities(config *types.Config) error {
    capabilities := config.Capabilities
    
    if len(capabilities) == 0 {
        // Drop all capabilities by default
        log.Println("Dropping all capabilities")
        if err := ClearAllCapabilities(); err != nil {
            return fmt.Errorf("failed to clear all capabilities: %v", err)
        }
        return nil
    }

    // Keep only specified capabilities
    log.Printf("Setting up capabilities: %v", capabilities)
    
    // First drop all capabilities
    if err := ClearAllCapabilities(); err != nil {
        return fmt.Errorf("failed to clear all capabilities: %v", err)
    }
    
    // Then add back the ones specified
    for _, cap := range capabilities {
        capValue, exists := types.CapMap[cap]
        if !exists {
            return fmt.Errorf("unknown capability: %s", cap)
        }
        
        if err := AddCapability(capValue); err != nil {
            return fmt.Errorf("failed to add capability %s: %v", cap, err)
        }
        log.Printf("Added capability: %s", cap)
    }
    
    return nil
}

// SetupCapabilitiesList provides an alternative function for direct capability list input
func SetupCapabilitiesList(capabilities []string) error {
    if len(capabilities) == 0 {
        // Drop all capabilities by default
        log.Println("Dropping all capabilities")
        if err := ClearAllCapabilities(); err != nil {
            return fmt.Errorf("failed to clear all capabilities: %v", err)
        }
        return nil
    }

    // Keep only specified capabilities
    log.Printf("Setting up capabilities: %v", capabilities)
    
    // First drop all capabilities
    if err := ClearAllCapabilities(); err != nil {
        return fmt.Errorf("failed to clear all capabilities: %v", err)
    }
    
    // Then add back the ones specified
    for _, cap := range capabilities {
        capValue, exists := types.CapMap[cap]
        if !exists {
            return fmt.Errorf("unknown capability: %s", cap)
        }
        
        if err := AddCapability(capValue); err != nil {
            return fmt.Errorf("failed to add capability %s: %v", cap, err)
        }
        log.Printf("Added capability: %s", cap)
    }
    
    return nil
}

// ClearAllCapabilities removes all capabilities from the current process
func ClearAllCapabilities() error {
    // Clear all ambient capabilities using direct prctl unix
    if _, _, errno := unix.Syscall6(unix.SYS_PRCTL, types.PR_CAP_AMBIENT, types.PR_CAP_AMBIENT_CLEAR_ALL, 0, 0, 0, 0); errno != 0 {
        return fmt.Errorf("failed to clear ambient capabilities: %v", errno)
    }
    
    // Clear bounding set capabilities
    for i := uintptr(0); i <= 40; i++ { // Loop through all possible capability values
        unix.Syscall6(unix.SYS_PRCTL, types.PR_CAPBSET_DROP, i, 0, 0, 0, 0)
    }
    
    return nil
}

// AddCapability adds a specific capability to the current process
func AddCapability(capValue uintptr) error {
    // Keep capabilities across setuid operations
    if _, _, errno := unix.Syscall6(unix.SYS_PRCTL, types.PR_SET_KEEPCAPS, 1, 0, 0, 0, 0); errno != 0 {
        return fmt.Errorf("failed to set PR_SET_KEEPCAPS: %v", errno)
    }

    // Get current capabilities
    header := types.CapUserHeader{
        Version: types.LINUX_CAPABILITY_VERSION_3,
        Pid:     0, // 0 means current process
    }
    var data [2]types.CapUserData
    if err := capget(&header, &data[0]); err != nil {
        return fmt.Errorf("failed to get current capabilities: %v", err)
    }

    // Calculate which data element and bit to set
    capIndex := capValue / 32
    capBit := uint32(1) << (capValue % 32)
    if capIndex >= 2 {
        return fmt.Errorf("capability value too large: %d", capValue)
    }

    // Set the capability in effective, permitted, and inheritable sets
    data[capIndex].Effective |= capBit
    data[capIndex].Permitted |= capBit
    data[capIndex].Inheritable |= capBit

    // Apply the new capabilities
    if err := capset(&header, &data[0]); err != nil {
        return fmt.Errorf("failed to set capabilities: %v", err)
    }

    // Set capability in the ambient set (inherited by child processes)
    // This must be done AFTER setting the capability in inheritable set
    if _, _, errno := unix.Syscall6(unix.SYS_PRCTL, types.PR_CAP_AMBIENT, types.PR_CAP_AMBIENT_RAISE, capValue, 0, 0, 0); errno != 0 {
        return fmt.Errorf("failed to add capability to ambient set: %v", errno)
    }

    return nil
}

// RemoveCapability removes a specific capability from the current process
func RemoveCapability(capValue uintptr) error {
    // Get current capabilities
    header := types.CapUserHeader{
        Version: types.LINUX_CAPABILITY_VERSION_3,
        Pid:     0,
    }
    var data [2]types.CapUserData
    if err := capget(&header, &data[0]); err != nil {
        return fmt.Errorf("failed to get current capabilities: %v", err)
    }

    // Calculate which data element and bit to clear
    capIndex := capValue / 32
    capBit := uint32(1) << (capValue % 32)
    if capIndex >= 2 {
        return fmt.Errorf("capability value too large: %d", capValue)
    }

    // Clear the capability from all sets
    data[capIndex].Effective &^= capBit
    data[capIndex].Permitted &^= capBit
    data[capIndex].Inheritable &^= capBit

    // Apply the changes
    if err := capset(&header, &data[0]); err != nil {
        return fmt.Errorf("failed to set capabilities: %v", err)
    }

    // Remove from ambient set
    if _, _, errno := unix.Syscall6(unix.SYS_PRCTL, types.PR_CAP_AMBIENT, types.PR_CAP_AMBIENT_LOWER, capValue, 0, 0, 0); errno != 0 {
        return fmt.Errorf("failed to remove capability from ambient set: %v", errno)
    }

    return nil
}

// GetCapabilities returns the current capabilities of the process
func GetCapabilities() (effective, permitted, inheritable uint64, err error) {
    header := types.CapUserHeader{
        Version: types.LINUX_CAPABILITY_VERSION_3,
        Pid:     0,
    }
    var data [2]types.CapUserData
    if err := capget(&header, &data[0]); err != nil {
        return 0, 0, 0, fmt.Errorf("failed to get current capabilities: %v", err)
    }

    // Combine the two 32-bit values into 64-bit values
    effective = uint64(data[1].Effective)<<32 | uint64(data[0].Effective)
    permitted = uint64(data[1].Permitted)<<32 | uint64(data[0].Permitted)
    inheritable = uint64(data[1].Inheritable)<<32 | uint64(data[0].Inheritable)

    return effective, permitted, inheritable, nil
}

// ValidateCapability checks if a capability name is valid
func ValidateCapability(capName string) bool {
    _, exists := types.CapMap[capName]
    return exists
}

// ListAvailableCapabilities returns all available capability names
func ListAvailableCapabilities() []string {
    var caps []string
    for capName := range types.CapMap {
        caps = append(caps, capName)
    }
    return caps
}