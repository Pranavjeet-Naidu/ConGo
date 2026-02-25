//go:build linux
// +build linux

package container

import (
	"congo/internals/types"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func RemoveContainer(containerID string) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is running
	if state.Status == "running" {
		return fmt.Errorf("cannot remove running container %s, stop it first", containerID)
	}

	// Remove container state file
	stateFile := filepath.Join(GetStateDir(), containerID+".json")
	if err := os.Remove(stateFile); err != nil {
		return fmt.Errorf("failed to remove container state file: %v", err)
	}

	// Clean up log directory if exists
	if state.LogDir != "" {
		if err := os.RemoveAll(state.LogDir); err != nil {
			log.Printf("Warning: failed to remove log directory: %v", err)
		}
	}

	return nil
}

func UpdateContainerResources(containerID, memory, cpu string, pids int) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container exists
	if state.Status == "" {
		return fmt.Errorf("container %s does not exist", containerID)
	}

	// Update memory limit if specified
	if memory != "" {
		memoryPath := filepath.Join("/sys/fs/cgroup/memory", "congo-"+containerID, "memory.limit_in_bytes")
		if err := os.WriteFile(memoryPath, []byte(memory), 0644); err != nil {
			return fmt.Errorf("failed to update memory limit: %v", err)
		}
		state.ResourceLimits.Memory = memory
	}

	// Update CPU shares if specified
	if cpu != "" {
		cpuPath := filepath.Join("/sys/fs/cgroup/cpu", "congo-"+containerID, "cpu.shares")
		if err := os.WriteFile(cpuPath, []byte(cpu), 0644); err != nil {
			return fmt.Errorf("failed to update CPU shares: %v", err)
		}
		state.ResourceLimits.CPU = cpu
	}

	// Update process limit if specified
	if pids > 0 {
		pidsPath := filepath.Join("/sys/fs/cgroup/pids", "congo-"+containerID, "pids.max")
		if err := os.WriteFile(pidsPath, []byte(strconv.Itoa(pids)), 0644); err != nil {
			return fmt.Errorf("failed to update process limit: %v", err)
		}
		state.ResourceLimits.ProcessLimit = pids
	}

	// Save updated state
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to save container state: %v", err)
	}

	return nil
}

func CommitContainer(containerID, imageName string) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check container status
	if state.Status == "running" {
		log.Printf("Warning: Committing a running container may result in inconsistent image")
	}

	// Create image directory
	imageDir := filepath.Join("/var/lib/congo/images", imageName)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return fmt.Errorf("failed to create image directory: %v", err)
	}

	// Create a tarball of the container filesystem
	tarPath := filepath.Join(imageDir, "rootfs.tar")
	tarCmd := exec.Command("tar", "-C", state.RootDir, "-cf", tarPath, ".")
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("failed to create image tarball: %v", err)
	}

	// Create image metadata file
	metadata := struct {
		ImageName   string            `json:"name"`
		CreatedAt   time.Time         `json:"created_at"`
		ContainerID string            `json:"container_id"`
		EnvVars     map[string]string `json:"env_vars"`
		Command     []string          `json:"command"`
	}{
		ImageName:   imageName,
		CreatedAt:   time.Now(),
		ContainerID: containerID,
		EnvVars:     state.EnvVars,
		Command:     state.Command,
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal image metadata: %v", err)
	}

	metadataPath := filepath.Join(imageDir, "metadata.json")
	if err := os.WriteFile(metadataPath, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write image metadata: %v", err)
	}

	return nil
}

func PauseContainer(containerID string) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is running
	if state.Status != "running" {
		return fmt.Errorf("container %s is not in running state", containerID)
	}

	// Create freezer cgroup directory if it doesn't exist
	freezerDir := filepath.Join("/sys/fs/cgroup/freezer", "congo-"+containerID)
	if err := os.MkdirAll(freezerDir, 0755); err != nil {
		return fmt.Errorf("failed to create freezer cgroup: %v", err)
	}

	// Add container process to the freezer cgroup
	if err := os.WriteFile(filepath.Join(freezerDir, "cgroup.procs"),
		[]byte(strconv.Itoa(state.Pid)), 0644); err != nil {
		return fmt.Errorf("failed to add process to freezer cgroup: %v", err)
	}

	// Freeze the container
	if err := os.WriteFile(filepath.Join(freezerDir, "freezer.state"),
		[]byte("FROZEN"), 0644); err != nil {
		return fmt.Errorf("failed to freeze container: %v", err)
	}

	// Update container state
	state.Status = "paused"
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to update container state: %v", err)
	}

	return nil
}

func UnpauseContainer(containerID string) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is paused
	if state.Status != "paused" {
		return fmt.Errorf("container %s is not in paused state", containerID)
	}

	// Path to container's freezer cgroup
	freezerDir := filepath.Join("/sys/fs/cgroup/freezer", "congo-"+containerID)

	// Unfreeze the container
	if err := os.WriteFile(filepath.Join(freezerDir, "freezer.state"),
		[]byte("THAWED"), 0644); err != nil {
		return fmt.Errorf("failed to unfreeze container: %v", err)
	}

	// Update container state
	state.Status = "running"
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to update container state: %v", err)
	}

	return nil
}

func AddVolumeToContainer(containerID, hostPath, containerPath string, readOnly bool) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is running
	if state.Status != "running" {
		return fmt.Errorf("container %s is not running", containerID)
	}

	// Create the mount point inside the container
	createDirCmd := fmt.Sprintf("mkdir -p %s", containerPath)
	if err := ExecInContainer(containerID, []string{"/bin/sh", "-c", createDirCmd}); err != nil {
		return fmt.Errorf("failed to create mount point: %v", err)
	}

	// Prepare nsenter command to enter container mount namespace
	args := []string{
		"-t", strconv.Itoa(state.Pid),
		"-m",
		"--",
	}

	// Use mount to attach the volume
	mountOpts := ""
	if readOnly {
		mountOpts = "-o ro"
	}

	mountCmd := fmt.Sprintf("mount %s --bind %s %s", mountOpts, hostPath, containerPath)
	args = append(args, "/bin/sh", "-c", mountCmd)

	cmd := exec.Command("nsenter", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount volume: %v", err)
	}

	// Update container state with the new mount
	newMount := types.Mount{
		Source:      hostPath,
		Destination: containerPath,
		ReadOnly:    readOnly,
	}

	state.Mounts = append(state.Mounts, newMount)

	// Save updated container state
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to update container state: %v", err)
	}

	return nil
}

func RemoveVolumeFromContainer(containerID, containerPath string) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is running
	if state.Status != "running" {
		return fmt.Errorf("container %s is not running", containerID)
	}

	// Check if the mount exists
	mountExists := false
	mountIndex := -1
	for i, mount := range state.Mounts {
		if mount.Destination == containerPath {
			mountExists = true
			mountIndex = i
			break
		}
	}

	if !mountExists {
		return fmt.Errorf("no mount found at path %s", containerPath)
	}

	// Prepare nsenter command to enter container mount namespace
	args := []string{
		"-t", strconv.Itoa(state.Pid),
		"-m",
		"--",
		"umount",
		containerPath,
	}

	cmd := exec.Command("nsenter", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to unmount volume: %v", err)
	}

	// Update container state by removing the mount
	state.Mounts = append(state.Mounts[:mountIndex], state.Mounts[mountIndex+1:]...)

	// Save updated container state
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to update container state: %v", err)
	}

	return nil
}

func StartContainer(containerID string, args []string) error {
	// Check if container exists
	stateFile := filepath.Join(GetStateDir(), containerID+".json")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		return fmt.Errorf("container %s does not exist", containerID)
	}

	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is already running
	if state.Status == "running" {
		return fmt.Errorf("container %s is already running", containerID)
	}

	// Reconstruct container configuration
	containerArgs := BuildArgsFromState(state)

	// Merge with any additional args provided
	if len(args) > 0 {
		// Override command if specified
		containerArgs = append(containerArgs[:len(containerArgs)-len(state.Command)], args...)
	}

	// Build command to start the container
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, containerArgs...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Set up namespaces using unix constants (from golang.org/x/sys/unix)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: unix.CLONE_NEWUTS |
			unix.CLONE_NEWPID |
			unix.CLONE_NEWNS |
			unix.CLONE_NEWNET |
			unix.CLONE_NEWIPC |
			unix.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      int(os.Getuid()),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      int(os.Getgid()),
				Size:        1,
			},
		},
		Unshareflags: unix.CLONE_NEWNS,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	// Update container state
	state.Pid = cmd.Process.Pid
	state.Status = "running"
	state.CreatedAt = time.Now()
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to save container state: %v", err)
	}

	if !state.Detached {
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("container process exited with error: %v", err)
		}
		// Update state after container exits
		state.Status = "stopped"
		state.Pid = 0
		if err := SaveContainerState(containerID, state); err != nil {
			return fmt.Errorf("failed to update container state: %v", err)
		}
	}

	return nil
}

func StopContainer(containerID string, force bool) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is running
	if state.Status != "running" {
		return fmt.Errorf("container %s is not running", containerID)
	}

	// Send signal to container process
	process, err := os.FindProcess(state.Pid)
	if err != nil {
		return fmt.Errorf("failed to find container process: %v", err)
	}

	// Send SIGTERM or SIGKILL
	signal := unix.SIGTERM
	if force {
		signal = unix.SIGKILL
	}

	if err := process.Signal(signal); err != nil {
		return fmt.Errorf("failed to send signal to container: %v", err)
	}

	// before: os.findprocess + process.wait(), issue is this doesn't work for non child processes on linux
	// so now better to go for polling instead 
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	exited := false
	for !exited {
		select {
		case <-timeout:
			if !force {
				return fmt.Errorf("container %s did not stop within timeout", containerID)
			}
			// Force kill
			if err := process.Kill(); err != nil {
				return fmt.Errorf("failed to kill container: %v", err)
			}
			exited = true
		case <-ticker.C:
			// Check if process is still running
			if _, err := process.Signal(syscall.Signal(0)); err != nil {
				exited = true
			}
		}
	}

	// Clean up network resources
	if err := CleanupContainerNetwork(state.Pid); err != nil {
		log.Printf("Warning: failed to clean up container network: %v", err)
	}

	// Clean up any port forwarding rules
	if err := CleanupPortForwarding(containerID); err != nil {
		log.Printf("Warning: failed to clean up port forwarding rules: %v", err)
	}

	// Update container state
	state.Status = "stopped"
	state.Pid = 0
	if err := SaveContainerState(containerID, state); err != nil {
		return fmt.Errorf("failed to update container state: %v", err)
	}

	return nil
}

func CleanupContainerNetwork(pid int) error {
	// Clean up veth pair - the host side only, container side vanishes with namespace
	hostVeth := fmt.Sprintf("hveth%d", pid)

	// Check if the interface exists before trying to remove it
	if _, err := net.InterfaceByName(hostVeth); err == nil {
		// Remove host veth interface
		if err := exec.Command("ip", "link", "del", hostVeth).Run(); err != nil {
			return fmt.Errorf("failed to remove host veth interface: %v", err)
		}
	}

	return nil
}

func CleanupPortForwarding(containerID string) error {
	// Get container state to find any port mappings
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state for port cleanup: %v", err)
	}

	// Check for network configuration in the state
	if state.Network.ContainerIP == "" || len(state.Network.PortMaps) == 0 {
		// No port mappings to clean up
		return nil
	}

	// Remove iptables rules for each port mapping
	for _, port := range state.Network.PortMaps {
		rule := fmt.Sprintf(
			"-t nat -D PREROUTING -p %s --dport %d -j DNAT --to-destination %s:%d",
			port.Protocol,
			port.HostPort,
			state.Network.ContainerIP,
			port.ContainerPort,
		)

		if err := exec.Command("iptables", strings.Split(rule, " ")...).Run(); err != nil {
			log.Printf("Warning: failed to remove port forwarding rule: %v", err)
		}

		// Also clean up any related MASQUERADE rules
		masqRule := fmt.Sprintf(
			"-t nat -D POSTROUTING -p %s -s %s --dport %d -j MASQUERADE",
			port.Protocol,
			state.Network.ContainerIP,
			port.ContainerPort,
		)

		if err := exec.Command("iptables", strings.Split(masqRule, " ")...).Run(); err != nil {
			log.Printf("Warning: failed to remove masquerade rule: %v", err)
		}
	}

	log.Printf("Cleaned up port forwarding rules for container %s", containerID)
	return nil
}

func RestartContainer(containerID string) error {
	// First stop the container
	if err := StopContainer(containerID, false); err != nil {
		return fmt.Errorf("failed to stop container for restart: %v", err)
	}

	// Load container state to get command arguments
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Reconstruct arguments needed to start the container
	args := BuildArgsFromState(state)

	// Start the container again
	if err := StartContainer(containerID, args); err != nil {
		return fmt.Errorf("failed to start container for restart: %v", err)
	}

	return nil
}

func ExecInContainer(containerID string, command []string) error {
	// Load container state
	state, err := LoadContainerState(containerID)
	if err != nil {
		return fmt.Errorf("failed to load container state: %v", err)
	}

	// Check if container is running
	if state.Status != "running" {
		return fmt.Errorf("container %s is not running", containerID)
	}

	// Prepare nsenter command to enter container namespaces
	args := []string{
		"-t", strconv.Itoa(state.Pid),
		"-m", "-u", "-i", "-n", "-p",
	}

	// Add the command to execute
	args = append(args, "--")
	args = append(args, command...)

	// Execute command inside container namespaces
	cmd := exec.Command("nsenter", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command in container: %v", err)
	}

	return nil
}

func GetStateDir() string {
	dir := "/var/run/congo"
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fallback to temp directory if can't create in /var/run
		dir = filepath.Join(os.TempDir(), "congo")
		os.MkdirAll(dir, 0755)
	}
	return dir
}

func SaveContainerState(containerID string, state types.ContainerState) error {
	stateDir := GetStateDir()
	stateFile := filepath.Join(stateDir, containerID+".json")

	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal container state: %v", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write container state file: %v", err)
	}

	return nil
}

func LoadContainerState(containerID string) (types.ContainerState, error) {
	stateDir := GetStateDir()
	stateFile := filepath.Join(stateDir, containerID+".json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return types.ContainerState{}, fmt.Errorf("failed to read container state file: %v", err)
	}

	var state types.ContainerState
	if err := json.Unmarshal(data, &state); err != nil {
		return types.ContainerState{}, fmt.Errorf("failed to unmarshal container state: %v", err)
	}

	return state, nil
}

func ListContainers() ([]types.ContainerState, error) {
	stateDir := GetStateDir()
	files, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.ContainerState{}, nil
		}
		return nil, fmt.Errorf("failed to read state directory: %v", err)
	}

	var containers []types.ContainerState
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			containerID := strings.TrimSuffix(file.Name(), ".json")
			state, err := LoadContainerState(containerID)
			if err != nil {
				log.Printf("Warning: failed to load state for container %s: %v", containerID, err)
				continue
			}
			containers = append(containers, state)
		}
	}

	return containers, nil
}

func BuildArgsFromState(state types.ContainerState) []string {
	// Create a basic set of environment variables
	args := []string{
		"/bin/sh",     // Default PATH
		"/root",       // Default HOME
		"root",        // Default USER
		"/bin/bash",   // Default SHELL
		"xterm",       // Default TERM
		"en_US.UTF-8", // Default LANG
	}

	// Add container rootfs option
	args = append(args, "--rootfs", state.RootDir)

	// Add container ID
	args = append(args, "--id", state.ID)

	// Add command separator
	args = append(args, "--")

	// Add the command from the state
	args = append(args, state.Command...)

	return args
}

func ParseMountSpec(spec string) (types.Mount, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return types.Mount{}, fmt.Errorf("invalid mount specification: %s", spec)
	}

	mount := types.Mount{
		Source:      parts[0],
		Destination: parts[1],
		ReadOnly:    len(parts) == 3 && parts[2] == "ro",
	}

	return mount, nil
}

func MustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Invalid number %s: %v", s, err)
	}
	return i
}
