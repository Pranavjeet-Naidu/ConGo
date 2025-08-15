package utils


import (
	
	"congo/congo/internals/types"
	"syscall"
	"unsafe"
)
// capget syscall wrapper
func capget(header *types.CapUserHeader, data *types.CapUserData) error {
    _, _, errno := syscall.Syscall(types.SYS_CAPGET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}

// capset syscall wrapper
func capset(header *types.CapUserHeader, data *types.CapUserData) error {
    _, _, errno := syscall.Syscall(types.SYS_CAPSET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}