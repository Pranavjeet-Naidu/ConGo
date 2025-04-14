# Summary of Functions in main_basic.go

## main()

- Entry point of the program.
- Checks the first command-line argument and calls either `run()` or `child()` based on the argument.
- Panics if the argument is not recognized.

## run()

- Prepares to execute the current program (exe) with the `child` argument and the rest of the command-line arguments.
- Sets up the command to use the same standard input, output, and error as the parent process.
- Sets `SysProcAttr` to create new namespaces for UTS, PID, and mount (NEWUTS, NEWPID, NEWNS).
- Calls `must(cmd.Run())` to run the command and handle any errors.

## child()

- Sets up the environment for the isolated command execution.
- Calls `cg()` to set up control groups (cgroups).
- Prepares to execute the command specified in the arguments.
- Checks if the root filesystem directory exists.
- Sets the hostname to "container".
- Changes the root directory to `ubuntufs` using `chroot`.
- Changes the working directory to `/`.
- Mounts the `proc` filesystem and a temporary filesystem.
- Runs the command.
- Unmounts the filesystems after the command execution.

## cg()

- Sets up a control group to limit the resources of the container.
- Creates a new directory for the control group.
- Limits the number of processes to 20.
- Sets up the control group to be removed after the container exits.
- Adds the current process to the control group.

## must(err error)

- Helper function to panic if an error occurs.
