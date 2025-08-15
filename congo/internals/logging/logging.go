package logging

import (
    "fmt"
    "os"
    "path/filepath"
    "syscall"
    "congo/congo/internals/container"
    "congo/congo/internals/types"
)

func setupLogging(config *types.Config) error {
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

func viewContainerLogs(containerID string) error {
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
