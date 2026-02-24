//go:build linux
// +build linux

package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
    "golang.org/x/sys/unix"
	"congo/internals/types"
)

func SetupLayeredRootfs(config *types.Config) error {
    // Create work and upper directories
    workDir := filepath.Join(os.TempDir(), "overlay-work")
    upperDir := config.Rootfs

    if err := os.MkdirAll(workDir, 0755); err != nil {
        return fmt.Errorf("failed to create work directory: %v", err)
    }

    if err := os.MkdirAll(upperDir, 0755); err != nil {
        return fmt.Errorf("failed to create upper directory: %v", err)
    }

    lowerDirs := strings.Join(config.ImageLayers, ":")
    overlayOpts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDirs, upperDir, workDir)

    if err := unix.Mount("overlay", "/", "overlay", 0, overlayOpts); err != nil {
        return fmt.Errorf("failed to mount overlay filesystem: %v", err)
    }

    return nil
}

func SetupRootfs(rootfs string) error {
    // Make root mount private
    if err := unix.Mount("", "/", "", unix.MS_REC|unix.MS_PRIVATE, ""); err != nil {
        return fmt.Errorf("error making root private: %v", err)
    }

    // Bind mount the new rootfs onto itself
    if err := unix.Mount(rootfs, rootfs, "", unix.MS_BIND|unix.MS_REC, ""); err != nil {
        return fmt.Errorf("error binding rootfs: %v", err)
    }

    // Create a directory for the old root inside the new rootfs
    pivotDir := filepath.Join(rootfs, ".pivot_root")
    if err := os.MkdirAll(pivotDir, 0755); err != nil {
        return fmt.Errorf("error creating pivot dir: %v", err)
    }

    // pivot_root: swap the root filesystem
    if err := unix.PivotRoot(rootfs, pivotDir); err != nil {
        return fmt.Errorf("error pivoting root: %v", err)
    }

    // Change directory to new root
    if err := os.Chdir("/"); err != nil {
        return fmt.Errorf("error chdir to new root: %v", err)
    }

    // Unmount the old root
    if err := unix.Unmount("/.pivot_root", unix.MNT_DETACH); err != nil {
        return fmt.Errorf("error unmounting old root: %v", err)
    }

    // Remove the old root directory
    if err := os.RemoveAll("/.pivot_root"); err != nil {
        return fmt.Errorf("error removing pivot dir: %v", err)
    }

    // Mount proc inside the new root
    if err := unix.Mount("proc", "/proc", "proc", 0, ""); err != nil {
        return fmt.Errorf("error mounting proc: %v", err)
    }

    return nil
}