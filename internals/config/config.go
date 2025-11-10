//go:build linux
// +build linux

package config

import (
	"fmt"
	"os/exec"
	"strconv"

	//"strings"

	"congo/internals/container"
	"congo/internals/types"
)

func ParseConfig(args []string, isChild bool) (*types.Config, error) {
	config := &types.Config{
		EnvVars:      make(map[string]string),
		Mounts:       make([]types.Mount, 0),
		Capabilities: make([]string, 0),
		Network: types.NetworkConfig{
			Bridge:   "congo0", // Default bridge name
			PortMaps: make([]types.PortMapping, 0),
		},
		LogConfig: types.LoggingConfig{
			EnableLogging: false,
			MaxLogSize:    10 * 1024 * 1024, // Default 10 MB
		},
		MonitorConfig: types.MonitoringConfig{
			Enabled:          false,
			Interval:         30, // Default 30 seconds
			MonitorCpu:       true,
			MonitorMemory:    true,
			MonitorProcesses: true,
		},
		Interactive: false,
		Detached:    false,
		StateDir:    container.GetStateDir(),
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
			mount := types.Mount{
				Source:      args[currentIdx+1],
				Destination: args[currentIdx+2],
				ReadOnly:    args[currentIdx+3] == "ro",
			}
			config.Mounts = append(config.Mounts, mount)
			currentIdx += 4
		case "--rootfs":
			if currentIdx+1 >= cmdIndex {
				return nil, fmt.Errorf("missing rootfs path")
			}
			config.Rootfs = args[currentIdx+1]
			currentIdx += 2
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

func ValidateConfig(config *types.Config) error {
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
