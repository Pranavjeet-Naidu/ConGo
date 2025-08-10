package types 

import (
	"fmt"
	"time"
	"syscall"
	"unsafe"
)

type CapUserHeader struct {
	Version uint32
	Pid int32 
}

type CapUserData struct {
	Effective uint32	
	Permitted uint32
	Inheritable uint32
}	

type LoggingConfig struct {
	LogDir string
	EnableLogging bool
	MaxLogSize int64
}

type MonitoringConfig struct {
	Enabled bool
	Interval int
	StatsFile string 
	MonitorCpu bool
	MonitorMemory bool
	MonitorProcesses bool
}

type Mount struct {
	Source string 
	Destination string 
	ReadOnly bool
}

type NetworkConfig struct {
	Bridge      string
	ContainerIP string
	PortMaps    []PortMapping
}

type Config struct {
    Rootfs       string
    ProcessLimit int
    MemoryLimit  string
    CpuShare     string
    EnvVars      map[string]string
    Command      []string
    Mounts       []Mount
    UseLayers    bool     
    ImageLayers  []string 
    User         string   
    Capabilities []string 
	Network NetworkConfig
	LogConfig LoggingConfig
    MonitorConfig MonitoringConfig
	ContainerID  string         
    State        ContainerState 
    Interactive  bool           
    Detached     bool           
    StateDir     string         
}

type PortMapping struct {
	HostPort      int
	ContainerPort int
	Protocol      string
}

// could have gone with flattened struct, but this allows for more consistent handling	
type ContainerState struct {
    ID           string            
    Pid          int               
    Status       string            
    CreatedAt    time.Time         
    Command      []string          
    RootDir      string            
    EnvVars      map[string]string 
    Mounts       []Mount           
    Interactive  bool              
    Detached     bool              
    LogDir       string            
    ResourceLimits struct {        
        Memory      string
        CPU         string
        ProcessLimit int
	}
    Network struct {               
        ContainerIP string
        Bridge      string
        PortMaps    []PortMapping
    }
}