# Summary of Functions in main.go

## main()

- Entry point of the program.
- Checks the first command-line argument and calls either `run()` or `child()` based on the argument.
- Panics if the argument is not recognized.

## run()

- Prepares to execute the current program (exe) with the `child` argument and the rest of the command-line arguments.
- Sets up the command to use the same standard input, output, and error as the parent process.
- Sets `SysProcAttr` to create new namespaces for UTS, PID, Network, Mount, and IPC.
- Calls `cmd.Run()` to run the command and handle any errors.

## child()

- Sets up the environment for the isolated command execution.
- Calls `setupContainer()` to set up the container environment.
- Prepares to execute the command specified in the arguments.
- Runs the command.

## parseConfig(args []string, isChild bool) (*Config, error)

- Parses command-line arguments to create a `Config` struct.
- Handles both parent and child process argument parsing.

## setupContainer(config *Config) error

- Sets the hostname to "container".
- Calls `setupRootfs()` to set up the root filesystem.
- Calls `setupCgroups()` to set up control groups.
- Sets environment variables.

## setupRootfs(rootfs string) error

- Mounts the root filesystem.
- Changes the root directory to the specified rootfs using `chroot`.
- Mounts necessary filesystems (`proc`, `tmpfs`).

## setupCgroups(config *Config) error

- Creates cgroup directories.
- Sets resource limits (process count, memory, CPU).
- Adds the current process to the cgroups.

## validateConfig(config *Config) error

- Validates the configuration parameters (rootfs path, process limit, memory limit, CPU share, command).

## mustAtoi(s string) int

- Converts a string to an integer and panics if an error occurs.

## parseEnvVars(envStr string) map[string]string

- Parses environment variables from a string.

## formatEnvVars(envMap map[string]string) string

- Formats environment variables into a string.
