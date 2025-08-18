//go:build linux
// +build linux

package logging

import (
    "fmt"
    "os"
    "path/filepath"
    "congo/congo/internals/container"
    //"congo/congo/internals/types"
)


func ViewContainerLogs(containerID string) error {
    // Load container state
    state, err := container.LoadContainerState(containerID)
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
