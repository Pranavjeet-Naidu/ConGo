# Functions Overview

## New and Modified Functions in step_3

### setupUser(user string) error
- Purpose: Drops privileges to specified user
- Parameters: username string
- Returns: error if user switch fails
- New in step_3

### main()
- Modified to include user namespace setup
- Added UID/GID mapping configuration
- Added CLONE_NEWUSER flag

### parseConfig()
- Added user parameter parsing
- Enhanced argument handling for user specification

### validateConfig()
- Added user validation
- Checks for su command availability

## Retained Functions from step_2

### setupContainer()
- Now includes user setup
- Retains all previous functionality

### setupCgroups()
- Unchanged from step_2
- Handles resource limitations

### setupLayeredRootfs()
- Unchanged from step_2
- Handles OverlayFS setup

### setupRootfs()
- Unchanged from step_2
- Handles basic root filesystem setup

### performBindMounts()
- Unchanged from step_2
- Handles bind mount setup

### cleanup()
- Unchanged from step_2
- Handles resource cleanup

### parseEnvVars() and formatEnvVars()
- Unchanged from step_2
- Handle environment variable processing

### parseMountSpec()
- Unchanged from step_2
- Parses mount specifications

## Function Call Flow

1. main()
2. parseConfig()
3. validateConfig()
4. setupContainer()
   - setupRootfs() or setupLayeredRootfs()
   - performBindMounts()
   - setupCgroups()
   - setupUser() (new)
5. cleanup()
