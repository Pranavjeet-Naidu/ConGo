package cgroups 

import(
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"congo/congo/internals/types"
)

func setupCgroups(config *types.Config) error {
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
