//go:build linux
// +build linux

package utils


import (
	"unsafe"
    
	"congo/congo/internals/types"
	"golang.org/x/sys/unix"
)
// capget syscall wrapper
func capget(header *types.CapUserHeader, data *types.CapUserData) error {
    _, _, errno := unix.Syscall(types.SYS_CAPGET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}

// capset syscall wrapper
func capset(header *types.CapUserHeader, data *types.CapUserData) error {
    _, _, errno := unix.Syscall(types.SYS_CAPSET, uintptr(unsafe.Pointer(header)), uintptr(unsafe.Pointer(data)), 0)
    if errno != 0 {
        return errno
    }
    return nil
}