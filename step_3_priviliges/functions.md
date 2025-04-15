## Privilege handling

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


