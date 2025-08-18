//go:build linux
// +build linux

package main

import (
	//"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	//"os/user"
	"path/filepath"
	"strconv"
	"strings"

	//"unsafe"
	"golang.org/x/sys/unix"
	//"net"
	"time"
	//"encoding/json"
	"congo/congo/internals/config"
	"congo/congo/internals/container"
	"congo/congo/internals/logging"
	"congo/congo/internals/types"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatalf("Usage: %s <command> [args...]", filepath.Base(os.Args[0]))
    }

    // Handle container lifecycle commands
    switch os.Args[1] {
    case "create":
        // Create a new container but don't start it
        config, err := config.ParseConfig(os.Args[2:], false)
        if err != nil {
            log.Fatalf("Error parsing config: %v", err)
        }
        
        // Generate a unique container ID if not provided
        if config.ContainerID == "" {
            config.ContainerID = fmt.Sprintf("congo-%d", time.Now().UnixNano())
        }
        
        // Initialize container state
        config.State = types.ContainerState{
            ID:        config.ContainerID,
            Status:    "created",
            CreatedAt: time.Now(),
            Command:   config.Command,
            RootDir:   config.Rootfs,
        }
        
        // Save the container state
        if err := container.SaveContainerState(config.ContainerID, config.State); err != nil {
            log.Fatalf("Error saving container state: %v", err)
        }
        
        fmt.Printf("Container created: %s\n", config.ContainerID)

	case "commit":
		// Create an image from a container
		if len(os.Args) < 4 {
			log.Fatalf("Usage: %s commit <container-id> <image-name>", os.Args[0])
		}
		containerID := os.Args[2]
		imageName := os.Args[3]
		
		if err := container.CommitContainer(containerID, imageName); err != nil {
			log.Fatalf("Error committing container: %v", err)
		}
		
		fmt.Printf("Container %s committed to image: %s\n", containerID, imageName)
	
	case "logs":
		// View container logs
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s logs <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := logging.ViewContainerLogs(containerID); err != nil {
			log.Fatalf("Error viewing container logs: %v", err)
		}
	
        
    case "start":
        // Start an existing container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s start <container-id>", os.Args[0])
        }
        containerID := os.Args[2]

        if err := container.StartContainer(containerID, os.Args[3:]); err != nil {
            log.Fatalf("Error starting container: %v", err)
        }
        
        fmt.Printf("Container started: %s\n", containerID)
    
	case "rm":
		// Remove a container
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s rm <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := container.RemoveContainer(containerID); err != nil {
			log.Fatalf("Error removing container: %v", err)
		}
		
		fmt.Printf("Container removed: %s\n", containerID)
	
    case "stop":
        // Stop a running container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s stop <container-id> [--force]", os.Args[0])
        }
        containerID := os.Args[2]
        force := len(os.Args) > 3 && os.Args[3] == "--force"
        
        if err := container.StopContainer(containerID, force); err != nil {
            log.Fatalf("Error stopping container: %v", err)
        }
        
        fmt.Printf("Container stopped: %s\n", containerID)
        
    case "restart":
        // Restart a container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s restart <container-id>", os.Args[0])
        }
        containerID := os.Args[2]

        if err := container.RestartContainer(containerID); err != nil {
            log.Fatalf("Error restarting container: %v", err)
        }
        
        fmt.Printf("Container restarted: %s\n", containerID)
        
    case "exec":
        // Execute a command in a running container
        if len(os.Args) < 4 {
            log.Fatalf("Usage: %s exec <container-id> <command> [args...]", os.Args[0])
        }
        containerID := os.Args[2]
        command := os.Args[3:]
        
        if err := container.ExecInContainer(containerID, command); err != nil {
            log.Fatalf("Error executing command in container: %v", err)
        }
        
    case "shell":
        // Start an interactive shell in a container
        if len(os.Args) < 3 {
            log.Fatalf("Usage: %s shell <container-id>", os.Args[0])
        }
        containerID := os.Args[2]
        
        // Default to bash, but fall back to sh if not available
        shell := []string{"/bin/bash"}
        if err := container.ExecInContainer(containerID, shell); err != nil {
            // Try sh if bash fails
            if err := container.ExecInContainer(containerID, []string{"/bin/sh"}); err != nil {
                log.Fatalf("Error starting shell in container: %v", err)
            }
        }
        
    case "ps":
        // List containers
        containers, err := container.ListContainers()
        if err != nil {
            log.Fatalf("Error listing containers: %v", err)
        }
        
        // Print container information
        fmt.Printf("%-20s %-10s %-20s %-30s\n", "CONTAINER ID", "STATUS", "CREATED", "COMMAND")
        for _, container := range containers {
            cmdStr := strings.Join(container.Command, " ")
            if len(cmdStr) > 30 {
                cmdStr = cmdStr[:27] + "..."
            }
            fmt.Printf("%-20s %-10s %-20s %-30s\n", 
                container.ID, 
                container.Status, 
                container.CreatedAt.Format(time.RFC3339), 
                cmdStr)
        }
        
    case "child":
        // Handle child process (container process)
        isChild := true
        config, err := config.ParseConfig(os.Args[2:], isChild)
        if err != nil {
            log.Fatalf("Error parsing config: %v", err)
        }

        if err := config.ValidateConfig(config); err != nil {
            log.Fatalf("Invalid config: %v", err)
        }

        if err := container.SetupContainer(config); err != nil {
            log.Fatalf("Error setting up container: %v", err)
        }

        // Check if interactive mode is requested
        if config.Interactive {
            // In interactive mode, start a shell
            if err := unix.Exec("/bin/bash", []string{"bash"}, os.Environ()); err != nil {
                // Try to fall back to sh if bash is not available
                if err := unix.Exec("/bin/sh", []string{"sh"}, os.Environ()); err != nil {
                    log.Fatalf("Error executing shell: %v", err)
                }
            }
        } else {
            // In non-interactive mode, execute the specified command
            if err := unix.Exec(config.Command[0], config.Command, os.Environ()); err != nil {
                log.Fatalf("Error executing command: %v", err)
            }
        }

	case "pause":
		// Pause a running container
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s pause <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := pauseContainer(containerID); err != nil {
			log.Fatalf("Error pausing container: %v", err)
		}
		
		fmt.Printf("Container paused: %s\n", containerID)
		
	case "unpause":
		// Unpause a paused container
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s unpause <container-id>", os.Args[0])
		}
		containerID := os.Args[2]
		
		if err := unpauseContainer(containerID); err != nil {
			log.Fatalf("Error unpausing container: %v", err)
		}
		
		fmt.Printf("Container unpaused: %s\n", containerID)
	case "update":
		// Update container resource limits
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s update <container-id> [--memory=<limit>] [--cpu=<shares>] [--pids=<limit>]", os.Args[0])
		}
		containerID := os.Args[2]
		
		var memory, cpu string
		var pids int
		
		// Parse update options
		for i := 3; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "--memory=") {
				memory = strings.TrimPrefix(os.Args[i], "--memory=")
			} else if strings.HasPrefix(os.Args[i], "--cpu=") {
				cpu = strings.TrimPrefix(os.Args[i], "--cpu=")
			} else if strings.HasPrefix(os.Args[i], "--pids=") {
				pidLimit := strings.TrimPrefix(os.Args[i], "--pids=")
				var err error
				pids, err = strconv.Atoi(pidLimit)
				if err != nil {
					log.Fatalf("Invalid pid limit: %v", err)
				}
			}
		}
		
		if err := updateContainerResources(containerID, memory, cpu, pids); err != nil {
			log.Fatalf("Error updating container resources: %v", err)
		}
		
		fmt.Printf("Container %s resources updated\n", containerID)
	
    case "volume-add":
		// Add a volume to a running container
		if len(os.Args) < 5 {
			log.Fatalf("Usage: %s volume-add <container-id> <host-path> <container-path> [ro]", os.Args[0])
		}
		containerID := os.Args[2]
		hostPath := os.Args[3]
		containerPath := os.Args[4]
		readOnly := len(os.Args) > 5 && os.Args[5] == "ro"
		
		if err := addVolumeToContainer(containerID, hostPath, containerPath, readOnly); err != nil {
			log.Fatalf("Error adding volume: %v", err)
		}
		
		fmt.Printf("Volume added to container %s: %s -> %s\n", containerID, hostPath, containerPath)
		
	case "volume-remove":
		// Remove a volume from a running container
		if len(os.Args) < 4 {
			log.Fatalf("Usage: %s volume-remove <container-id> <container-path>", os.Args[0])
		}
		containerID := os.Args[2]
		containerPath := os.Args[3]
		
		if err := removeVolumeFromContainer(containerID, containerPath); err != nil {
			log.Fatalf("Error removing volume: %v", err)
		}
		
		fmt.Printf("Volume removed from container %s: %s\n", containerID, containerPath)
	 
    case "run":
		
        // Create and start a container in one step
        config, err := parseConfig(os.Args[2:], false)
        if err != nil {
            log.Fatalf("Error parsing config: %v", err)
        }
        config.State.Network.ContainerIP = config.Network.ContainerIP
		config.State.Network.Bridge = config.Network.Bridge
		config.State.Network.PortMaps = config.Network.PortMaps
        if err := validateConfig(config); err != nil {
            log.Fatalf("Invalid config: %v", err)
        }
        
        // Generate a unique container ID
        if config.ContainerID == "" {
            config.ContainerID = fmt.Sprintf("congo-%d", time.Now().UnixNano())
        }
        
        // Initialize container state
        config.State = ContainerState{
            ID:        config.ContainerID,
            Status:    "created", // Will be updated to "running" when started
            CreatedAt: time.Now(),
            Command:   config.Command,
            RootDir:   config.Rootfs,
        }
        
        // Save the container state
        if err := saveContainerState(config.ContainerID, config.State); err != nil {
            log.Fatalf("Error saving container state: %v", err)
        }
        
        // Start the container
        cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)
        cmd.Stdin = os.Stdin
        cmd.Stdout = os.Stdout
        cmd.Stderr = os.Stderr
        cmd.SysProcAttr = &unix.SysProcAttr{
            Cloneflags: unix.CLONE_NEWUTS |
                unix.CLONE_NEWPID |
                unix.CLONE_NEWNS |
                unix.CLONE_NEWNET |
                unix.CLONE_NEWIPC |
                unix.CLONE_NEWUSER,
            UidMappings: []unix.SysProcIDMap{
                {
                    ContainerID: 0,
                    HostID:     os.Getuid(),
                    Size:       1,
                },
            },
            GidMappings: []unix.SysProcIDMap{
                {
                    ContainerID: 0,
                    HostID:     os.Getgid(),
                    Size:       1,
                },
            },
            Unshareflags: unix.CLONE_NEWNS,
        }
        
        if err := cmd.Start(); err != nil {
            log.Fatalf("Error starting container: %v", err)
        }
        
        // Update container state
        config.State.Status = "running"
        config.State.Pid = cmd.Process.Pid
        if err := saveContainerState(config.ContainerID, config.State); err != nil {
            log.Printf("Warning: failed to update container state: %v", err)
        }
        
        fmt.Printf("Container started: %s\n", config.ContainerID)
        
        // If not detached, wait for the container to exit
        if !config.Detached {
            if err := cmd.Wait(); err != nil {
                log.Printf("Container exited with error: %v", err)
            }
            
            // Update container state to stopped
            config.State.Status = "stopped"
            config.State.Pid = 0
            if err := saveContainerState(config.ContainerID, config.State); err != nil {
                log.Printf("Warning: failed to update container state: %v", err)
            }
        }
        
    default:
        log.Fatalf("Unknown command: %s", os.Args[1])
    }
}


