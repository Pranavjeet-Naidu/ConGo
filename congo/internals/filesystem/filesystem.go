package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"congo/congo/internals/types"
)

func setupLayeredRootfs(config *types.Config) error {
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

    if err := syscall.Mount("overlay", "/", "overlay", 0, overlayOpts); err != nil {
        return fmt.Errorf("failed to mount overlay filesystem: %v", err)
    }

    return nil
}

func setupRootfs(rootfs string) error {
    if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
        return fmt.Errorf("error making root private: %v", err)
    }

    if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
        return fmt.Errorf("error binding rootfs: %v", err)
    }

    if err := os.Chdir(rootfs); err != nil {
        return fmt.Errorf("error changing directory to rootfs: %v", err)
    }

    if err := syscall.Mount("proc", "proc", "proc", 0, ""); err != nil {
        return fmt.Errorf("error mounting proc: %v", err)
    }

    return nil
}
