package monitoring

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
    
    "congo/congo/internals/types"
)


func startResourceMonitoring(config *types.Config) error {
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