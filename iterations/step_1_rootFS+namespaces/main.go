package main

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
)

// Config stores container configuration
type Config struct {
    Rootfs       string
    ProcessLimit int
    MemoryLimit  string
    CpuShare     string
    EnvVars      map[string]string
    Command      []string
}

func main() {
    log.SetPrefix("container-runtime: ")
    if len(os.Args) < 2 {
        log.Fatal("Usage: go run main.go run <rootfs> <process_limit> <memory_limit> <cpu_share> <env_vars> -- <cmd> <args>")
    }

    switch os.Args[1] {
    case "run":
        if err := run(); err != nil {
            log.Printf("%v", err)
            os.Exit(1)
        }
    case "child":
        if err := child(); err != nil {
            log.Printf("%v", err)
            os.Exit(1)
        }
    default:
        log.Fatal("Unknown command. Use 'run' or 'child'")
    }
}

func parseConfig(args []string, isChild bool) (*Config, error) {
    var cmdIndex int
    var configArgs []string

    if isChild {
        // For child process, command starts after the env vars
        configArgs = args[2:7] // child process args: [child, rootfs, limit, memory, cpu, env]
        cmdArgs := args[7:]    // remaining args are the command
        return &Config{
            Rootfs:       configArgs[0],
            ProcessLimit: mustAtoi(configArgs[1]),
            MemoryLimit:  configArgs[2],
            CpuShare:     configArgs[3],
            EnvVars:      parseEnvVars(configArgs[4]),
            Command:      cmdArgs,
        }, nil
    }

    // For parent process, look for -- separator
    for i, arg := range args {
        if arg == "--" {
            cmdIndex = i
            break
        }
    }

    if cmdIndex == 0 {
        return nil, fmt.Errorf("command separator '--' not found")
    }

    if cmdIndex < 7 { // "run" + 5 required args + "--"
        return nil, fmt.Errorf("insufficient arguments before '--'")
    }

    return &Config{
        Rootfs:       args[2],
        ProcessLimit: mustAtoi(args[3]),
        MemoryLimit:  args[4],
        CpuShare:     args[5],
        EnvVars:      parseEnvVars(args[6]),
        Command:      args[cmdIndex+1:],
    }, nil
}

func mustAtoi(s string) int {
    i, err := strconv.Atoi(s)
    if err != nil {
        log.Fatalf("Invalid number %s: %v", s, err)
    }
    return i
}

func run() error {
    config, err := parseConfig(os.Args, false)
    if err != nil {
        return fmt.Errorf("error parsing config: %v", err)
    }

    if err := validateConfig(config); err != nil {
        return fmt.Errorf("invalid configuration: %v", err)
    }

    log.Printf("Running %v in isolated environment\n", config.Command)

    args := []string{
        "child",
        config.Rootfs,
        strconv.Itoa(config.ProcessLimit),
        config.MemoryLimit,
        config.CpuShare,
        formatEnvVars(config.EnvVars),
    }
    args = append(args, config.Command...)

    cmd := exec.Command("/proc/self/exe", args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Modified clone flags
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Cloneflags: syscall.CLONE_NEWUTS |
            syscall.CLONE_NEWPID |
            syscall.CLONE_NEWNS |
            syscall.CLONE_NEWNET |
            syscall.CLONE_NEWIPC,
        
        Unshareflags: syscall.CLONE_NEWNS,
        
    }

    return cmd.Run()
}
func child() error {
    config, err := parseConfig(os.Args, true)
    if err != nil {
        return fmt.Errorf("error parsing config: %v", err)
    }

    log.Printf("Setting up container environment\n")

    if err := setupContainer(config); err != nil {
        return fmt.Errorf("error setting up container: %v", err)
    }

    cmd := exec.Command(config.Command[0], config.Command[1:]...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Run()
}

func setupContainer(config *Config) error {
	// Set hostname
	if err := syscall.Sethostname([]byte("container")); err != nil {
		return fmt.Errorf("error setting hostname: %v", err)
	}

	// Setup root filesystem
	if err := setupRootfs(config.Rootfs); err != nil {
		return fmt.Errorf("error setting up rootfs: %v", err)
	}

	// Setup cgroups
	if err := setupCgroups(config); err != nil {
		return fmt.Errorf("error setting up cgroups: %v", err)
	}

	// Set environment variables
	for k, v := range config.EnvVars {
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("error setting environment variable %s: %v", k, err)
		}
	}

	return nil
}

func setupRootfs(rootfs string) error {
    // First mount the rootfs to itself as private to prevent mount propagation
    if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
        return fmt.Errorf("error making root private: %v", err)
    }

    // Pivot into the new root
    if err := syscall.Chroot(rootfs); err != nil {
        return fmt.Errorf("chroot error: %v", err)
    }

    if err := os.Chdir("/"); err != nil {
        return fmt.Errorf("chdir error: %v", err)
    }

    // Create necessary directories if they don't exist
    dirs := []string{"/proc", "/tmp", "/var/tmp"}
    for _, dir := range dirs {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("error creating directory %s: %v", dir, err)
        }
    }

    // Mount proc with proper flags
    if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
        return fmt.Errorf("error mounting proc: %v", err)
    }

    // Mount tmpfs with proper permissions
    if err := syscall.Mount("tmpfs", "/tmp", "tmpfs", 0, ""); err != nil {
        return fmt.Errorf("error mounting /tmp: %v", err)
    }

    if err := syscall.Mount("tmpfs", "/var/tmp", "tmpfs", 0, ""); err != nil {
        return fmt.Errorf("error mounting /var/tmp: %v", err)
    }

    return nil
}


func setupCgroups(config *Config) error {
	containerId := fmt.Sprintf("container-%d", os.Getpid())
	cgroupPaths := map[string]string{
		"pids":    filepath.Join("/sys/fs/cgroup/pids", containerId),
		"memory":  filepath.Join("/sys/fs/cgroup/memory", containerId),
		"cpu":     filepath.Join("/sys/fs/cgroup/cpu", containerId),
		"blkio":   filepath.Join("/sys/fs/cgroup/blkio", containerId),
	}

	// Create cgroup directories
	for _, path := range cgroupPaths {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	// Set resource limits
	limits := map[string][]byte{
		filepath.Join(cgroupPaths["pids"], "pids.max"):                []byte(strconv.Itoa(config.ProcessLimit)),
		filepath.Join(cgroupPaths["memory"], "memory.limit_in_bytes"): []byte(config.MemoryLimit),
		filepath.Join(cgroupPaths["cpu"], "cpu.shares"):               []byte(config.CpuShare),
		filepath.Join(cgroupPaths["blkio"], "blkio.weight"):          []byte("100"),
	}

	for path, value := range limits {
		if err := os.WriteFile(path, value, 0644); err != nil {
			return fmt.Errorf("error writing to %s: %v", path, err)
		}
	}

	// Add process to cgroups
	pid := []byte(strconv.Itoa(os.Getpid()))
	for _, path := range cgroupPaths {
		if err := os.WriteFile(filepath.Join(path, "cgroup.procs"), pid, 0644); err != nil {
			return fmt.Errorf("error writing to cgroup.procs: %v", err)
		}
	}

	return nil
}

func validateConfig(config *Config) error {
	// Validate rootfs
	if stat, err := os.Stat(config.Rootfs); err != nil || !stat.IsDir() {
		return fmt.Errorf("invalid rootfs path: %s", config.Rootfs)
	}

	// Validate process limit
	if config.ProcessLimit <= 0 {
		return fmt.Errorf("process limit must be positive")
	}

	// Validate memory limit format
	if _, err := strconv.ParseInt(strings.TrimSuffix(config.MemoryLimit, "m"), 10, 64); err != nil {
		return fmt.Errorf("invalid memory limit format: %s", config.MemoryLimit)
	}

	// Validate CPU share
	if _, err := strconv.Atoi(config.CpuShare); err != nil {
		return fmt.Errorf("invalid CPU share: %s", config.CpuShare)
	}

	// Validate command
	if len(config.Command) == 0 {
		return fmt.Errorf("command cannot be empty")
	}

	return nil
}

func parseEnvVars(envStr string) map[string]string {
	envMap := make(map[string]string)
	pairs := strings.Split(envStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}
	return envMap
}

func formatEnvVars(envMap map[string]string) string {
	pairs := make([]string, 0, len(envMap))
	for k, v := range envMap {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ",")
}