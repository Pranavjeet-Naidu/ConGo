//go:build linux
// +build linux

package utils

import (
	"congo/internals/types"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"golang.org/x/sys/unix"
	"strings"
)

// getHomeDirectory determines the appropriate home directory
func GetHomeDirectory(uid int, username string) string {
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

func Cleanup(config *types.Config) error {
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
        if err := unix.Unmount(mount.Destination, 0); err != nil {
            return fmt.Errorf("failed to unmount: %v", err)
        }
    }
    return nil
}

func ParseEnvVars(envStr string) map[string]string {
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

func FormatEnvVars(envMap map[string]string) string {
    var envStrs []string
    for k, v := range envMap {
        envStrs = append(envStrs, fmt.Sprintf("%s=%s", k, v))
    }
    return strings.Join(envStrs, ",")
}