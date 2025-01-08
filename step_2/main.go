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

// Mount represents a bind mount configuration
type Mount struct {
    Source      string
    Destination string
    ReadOnly    bool
}

// Config stores container configuration
type Config struct {
    Rootfs       string
    ProcessLimit int
    MemoryLimit  string
    CpuShare     string
    EnvVars      map[string]string
    Command      []string
    Mounts       []Mount
    UseLayers    bool     // Whether to use OverlayFS
    ImageLayers  []string // Paths to lower layers for OverlayFS
}

func main() {
    log.SetPrefix("container-runtime: ")
    if len(os.Args) < 2 {
        log.Fatal("Usage: go run main.go run <rootfs> <process_limit> <memory_limit> <cpu_share> <env_vars> [--mount source:dest[:ro]] [--layers layer1,layer2] -- <cmd> <args>")
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

func mustAtoi(s string) int {
    i, err := strconv.Atoi(s)
    if err != nil {
        log.Fatalf("Invalid number %s: %v", s, err)
    }
    return i
}

func parseMountSpec(spec string) (Mount, error) {
    parts := strings.Split(spec, ":")
    if len(parts) < 2 || len(parts) > 3 {
        return Mount{}, fmt.Errorf("invalid mount specification: %s", spec)
    }

    mount := Mount{
        Source:      parts[0],
        Destination: parts[1],
        ReadOnly:    len(parts) == 3 && parts[2] == "ro",
    }

    return mount, nil
}

func parseConfig(args []string, isChild bool) (*Config, error) {
    config := &Config{
        EnvVars: make(map[string]string),
        Mounts:  make([]Mount, 0),
    }

    if isChild {
        // Parse child process args
        config.Rootfs = args[2]
        config.ProcessLimit = mustAtoi(args[3])
        config.MemoryLimit = args[4]
        config.CpuShare = args[5]
        config.EnvVars = parseEnvVars(args[6])

        // Parse mounts
        mountCount := mustAtoi(args[7])
        currentIdx := 8
        for i := 0; i < mountCount; i++ {
            mount, err := parseMountSpec(args[currentIdx])
            if err != nil {
                return nil, err
            }
            config.Mounts = append(config.Mounts, mount)
            currentIdx++
        }

        // Parse layers
        config.UseLayers = args[currentIdx] == "true"
        currentIdx++
        if config.UseLayers {
            config.ImageLayers = strings.Split(args[currentIdx], ",")
            currentIdx++
        }

        config.Command = args[currentIdx:]
        return config, nil
    }

    // Parse parent process args
    var cmdIndex int
    for i, arg := range args {
        if arg == "--" {
            cmdIndex = i
            break
        }
    }

    if cmdIndex == 0 {
        return nil, fmt.Errorf("command separator '--' not found")
    }

    // Parse basic configuration
    config.Rootfs = args[2]
    config.ProcessLimit = mustAtoi(args[3])
    config.MemoryLimit = args[4]
    config.CpuShare = args[5]
    config.EnvVars = parseEnvVars(args[6])

    // Parse additional arguments before --
    currentIdx := 7
    for currentIdx < cmdIndex {
        switch args[currentIdx] {
        case "--mount":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing mount specification")
            }
            mount, err := parseMountSpec(args[currentIdx+1])
            if err != nil {
                return nil, err
            }
            config.Mounts = append(config.Mounts, mount)
            currentIdx += 2
        case "--layers":
            if currentIdx+1 >= cmdIndex {
                return nil, fmt.Errorf("missing layer specification")
            }
            config.UseLayers = true
            config.ImageLayers = strings.Split(args[currentIdx+1], ",")
            currentIdx += 2
        default:
            return nil, fmt.Errorf("unknown option: %s", args[currentIdx])
        }
    }

    config.Command = args[cmdIndex+1:]
    return config, nil
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

    // Prepare child process arguments
    args := []string{
        "child",
        config.Rootfs,
        strconv.Itoa(config.ProcessLimit),
        config.MemoryLimit,
        config.CpuShare,
        formatEnvVars(config.EnvVars),
        strconv.Itoa(len(config.Mounts)),
    }

    // Add mount specifications
    for _, mount := range config.Mounts {
        mountSpec := fmt.Sprintf("%s:%s", mount.Source, mount.Destination)
        if mount.ReadOnly {
            mountSpec += ":ro"
        }
        args = append(args, mountSpec)
    }

    // Add layer information
    args = append(args, strconv.FormatBool(config.UseLayers))
    if config.UseLayers {
        args = append(args, strings.Join(config.ImageLayers, ","))
    }

    // Add command
    args = append(args, config.Command...)

    cmd := exec.Command("/proc/self/exe", args...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
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
    defer func() {
        if err := cleanup(config); err != nil {
            log.Printf("Warning: cleanup failed: %v", err)
        }
    }()

    if err := syscall.Sethostname([]byte("container")); err != nil {
        return fmt.Errorf("error setting hostname: %v", err)
    }

    if config.UseLayers {
        if err := setupLayeredRootfs(config); err != nil {
            return fmt.Errorf("error setting up layered rootfs: %v", err)
        }
    } else {
        if err := setupRootfs(config.Rootfs); err != nil {
            return fmt.Errorf("error setting up rootfs: %v", err)
        }
    }

    if err := performBindMounts(config.Mounts); err != nil {
        return fmt.Errorf("error performing bind mounts: %v", err)
    }

    if err := setupCgroups(config); err != nil {
        return fmt.Errorf("error setting up cgroups: %v", err)
    }

    for k, v := range config.EnvVars {
        if err := os.Setenv(k, v); err != nil {
            return fmt.Errorf("error setting environment variable %s: %v", k, err)
        }
    }

    return nil
}

func setupLayeredRootfs(config *Config) error {
    // Create work and upper directories
    workDir := filepath.Join(os.TempDir(), "overlay-work")
    upperDir := config.Rootfs

    if err := os.MkdirAll(workDir, 0755); err != nil {
        return fmt.Errorf("failed to create overlay work directory: %v", err)
    }

    // Prepare lower directory string (image layers)
    lowerDirs := strings.Join(config.ImageLayers, ":")

    // Mount overlay filesystem
    opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDirs, upperDir, workDir)
    if err := syscall.Mount("overlay", upperDir, "overlay", 0, opts); err != nil {
        return fmt.Errorf("failed to mount overlay filesystem: %v", err)
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

func performBindMounts(mounts []Mount) error {
    for _, mount := range mounts {
        flags := syscall.MS_BIND
        if mount.ReadOnly {
            flags |= syscall.MS_RDONLY
        }

        if err := os.MkdirAll(mount.Destination, 0755); err != nil {
            return fmt.Errorf("failed to create mount point %s: %v", mount.Destination, err)
        }

        if err := syscall.Mount(mount.Source, mount.Destination, "", uintptr(flags), ""); err != nil {
            return fmt.Errorf("failed to bind mount %s to %s: %v", mount.Source, mount.Destination, err)
        }
    }
    return nil
}

func setupCgroups(config *Config) error {
    containerId := fmt.Sprintf("container-%d", os.Getpid())
    cgroupPaths := map[string]string{
        "pids":   filepath.Join("/sys/fs/cgroup/pids", containerId),
        "memory": filepath.Join("/sys/fs/cgroup/memory", containerId),
        "cpu":    filepath.Join("/sys/fs/cgroup/cpu", containerId),
        "blkio":  filepath.Join("/sys/fs/cgroup/blkio", containerId),
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
        filepath.Join(cgroupPaths["blkio"], "blkio.weight"):           []byte("100"),
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

func cleanup(config *Config) error {
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
            log.Printf("Warning: failed to remove cgroup path %s: %v", path, err)
        }
    }

    // Unmount OverlayFS if used
    if config.UseLayers {
        if err := syscall.Unmount(config.Rootfs, 0); err != nil {
            log.Printf("Warning: failed to unmount overlay filesystem: %v", err)
        }
    }

    // Unmount /proc and temporary filesystems
    syscall.Unmount("/proc", 0)
    syscall.Unmount("/tmp", 0)
    syscall.Unmount("/var/tmp", 0)

    return nil
}

// parseEnvVars parses environment variables from a comma-separated string
func parseEnvVars(envStr string) map[string]string {
    envMap := make(map[string]string)
    if envStr == "" {
        return envMap
    }

    pairs := strings.Split(envStr, ",")
    for _, pair := range pairs {
        kv := strings.SplitN(pair, "=", 2)
        if len(kv) == 2 {
            // Trim any whitespace and quotes
            key := strings.TrimSpace(kv[0])
            value := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
            if key != "" {
                envMap[key] = value
            }
        }
    }
    return envMap
}

// formatEnvVars formats environment variables map into a comma-separated string
func formatEnvVars(envMap map[string]string) string {
    if len(envMap) == 0 {
        return ""
    }

    pairs := make([]string, 0, len(envMap))
    for k, v := range envMap {
        // Escape commas and equals signs in values
        escapedValue := strings.ReplaceAll(strings.ReplaceAll(v, ",", "\\,"), "=", "\\=")
        pairs = append(pairs, fmt.Sprintf("%s=%s", k, escapedValue))
    }
    return strings.Join(pairs, ",")
}

// Validate config to ensure everything is properly set
func validateConfig(config *Config) error {
    if config == nil {
        return fmt.Errorf("config cannot be nil")
    }

    if stat, err := os.Stat(config.Rootfs); err != nil || !stat.IsDir() {
        return fmt.Errorf("invalid rootfs path: %s", config.Rootfs)
    }

    if config.ProcessLimit <= 0 {
        return fmt.Errorf("process limit must be positive")
    }

    memoryLimit := strings.TrimSuffix(config.MemoryLimit, "m")
    if _, err := strconv.ParseInt(memoryLimit, 10, 64); err != nil {
        return fmt.Errorf("invalid memory limit format: %s", config.MemoryLimit)
    }

    if cpuShare, err := strconv.Atoi(config.CpuShare); err != nil || cpuShare < 2 || cpuShare > 262144 {
        return fmt.Errorf("invalid CPU share: %s (must be between 2 and 262144)", config.CpuShare)
    }

    if len(config.Command) == 0 {
        return fmt.Errorf("command cannot be empty")
    }

    for _, mount := range config.Mounts {
        if stat, err := os.Stat(mount.Source); err != nil {
            return fmt.Errorf("invalid mount source path: %s", mount.Source)
        } else if !stat.IsDir() && mount.ReadOnly {
            // Allow non-directory mounts only for read-only binds
            return fmt.Errorf("non-directory mount source must be read-only: %s", mount.Source)
        }
    }

    if config.UseLayers {
        for _, layer := range config.ImageLayers {
            if stat, err := os.Stat(layer); err != nil || !stat.IsDir() {
                return fmt.Errorf("invalid layer path: %s (must be a directory)", layer)
            }
        }
    }

    return nil
}