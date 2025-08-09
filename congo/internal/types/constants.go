package types 

// Linux capability constants and types (missing from unix package on some platforms)
const (
    // prctl() constants
	PR_CAP_AMBIENT_RAISE    = 2
    PR_CAP_AMBIENT_LOWER    = 3
    PR_CAP_AMBIENT_CLEAR_ALL = 4
	PR_GET_KEEPCAPS		     = 7
    PR_SET_KEEPCAPS          = 8
    PR_CAPBSET_DROP         = 24
	PR_GET_SECUREBITS       = 27
	PR_SET_SECUREBITS       = 28
    PR_CAP_AMBIENT          = 47
    
    // Linux capability version
    LINUX_CAPABILITY_VERSION_3 = 0x20080522
)


var CapMap = map[string]uintptr{
		"CAP_CHOWN":            0,
        "CAP_DAC_OVERRIDE":     1,
        "CAP_DAC_READ_SEARCH":  2,
        "CAP_FOWNER":           3,
        "CAP_FSETID":           4,
        "CAP_KILL":             5,
        "CAP_SETGID":           6,
        "CAP_SETUID":           7,
        "CAP_SETPCAP":          8,
        "CAP_NET_BIND_SERVICE": 10,
        "CAP_NET_RAW":          13,
        "CAP_SYS_CHROOT":       18,
        "CAP_MKNOD":            27,
        "CAP_AUDIT_WRITE":      29,
        "CAP_SETFCAP":          31,
}

// Container status constants 
const(
	StatusCreated = "created"
	StatusRunning = "running"
	StatusStopped = "stopped"
	StatusPaused = "paused"
	StatusExited = "exited"
)

// Default container settings 
const (
	DefaultBridgeName = "congo0"
	DefaultStateDir = "/var/run/congo"
	DefaultLogDir = "/var/log/congo"
	DefaultMaxLogSize = 10 * 1024 * 1024
	DefaultMonitorInterval = 30
	DefaultImageDir = "var/lib/congo/images"
)

// Network defaults 
const (
	DefaultSubnet = "172.20.0.0/16"
	DefaultGateway = "172.20.0.1"
)

//Timeout constants 
const(
	ContainerStopTimeout = 10
)

// File permissions
const(
	StateFileMode = 0644
	LogFileMode = 0644
	DirMode = 0755
)

// Cgroup paths 
const (
	CgroupV1Base = "/sys/fs/cgroup"
	CgroupV2Base = "/sys/fs/cgroup"
)