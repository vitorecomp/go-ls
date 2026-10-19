package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// DetectComputerID returns a persistent identifier for the current computer by reading
// /etc/machine-id (or /var/lib/dbus/machine-id), falling back to the provided hostname
// or os.Hostname().
func DetectComputerID(fallbackHostname string) string {
	candidates := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			id := strings.TrimSpace(string(data))
			if id != "" {
				return id
			}
		}
	}

	if fallbackHostname != "" {
		return fallbackHostname
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "unknown-computer"
}

// DetectDriveID returns a unique identifier for the underlying hard drive / filesystem
// volume that stores targetPath.
//
// Resolution order on Linux/Unix:
//  1. Filesystem UUID from /dev/disk/by-uuid matching the directory's block device number (stat.Dev).
//  2. Partition UUID from /dev/disk/by-partuuid matching stat.Dev.
//  3. Hardware device ID from /dev/disk/by-id matching stat.Dev.
//  4. Kernel 64-bit filesystem identifier (statfs.Fsid).
//  5. Kernel device major:minor number from stat.Dev.
func DetectDriveID(targetPath string) string {
	if targetPath == "" {
		targetPath = "."
	}
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		absPath = filepath.Clean(targetPath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		// Try parent directory if targetPath does not exist yet.
		info, err = os.Stat(filepath.Dir(absPath))
		if err != nil {
			return "unknown-drive"
		}
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if ok && stat != nil {
		dev := uint64(stat.Dev)

		if uuid := matchDiskByDir("/dev/disk/by-uuid", dev); uuid != "" {
			return "uuid:" + uuid
		}
		if partUUID := matchDiskByDir("/dev/disk/by-partuuid", dev); partUUID != "" {
			return "partuuid:" + partUUID
		}
		if diskID := matchDiskByDir("/dev/disk/by-id", dev); diskID != "" {
			return "diskid:" + diskID
		}

		var stfs syscall.Statfs_t
		if err := syscall.Statfs(absPath, &stfs); err == nil {
			v0 := uint32(stfs.Fsid.X__val[0])
			v1 := uint32(stfs.Fsid.X__val[1])
			if v0 != 0 || v1 != 0 {
				return fmt.Sprintf("fsid:%08x-%08x", v0, v1)
			}
		}

		major, minor := deviceMajorMinor(dev)
		return fmt.Sprintf("dev:%d:%d", major, minor)
	}

	return "unknown-drive"
}

func matchDiskByDir(byDir string, targetDev uint64) string {
	entries, err := os.ReadDir(byDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		fullPath := filepath.Join(byDir, entry.Name())
		// os.Stat follows the symlink to the block device node (e.g. /dev/sda1 or /dev/dm-1).
		devInfo, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		devStat, ok := devInfo.Sys().(*syscall.Stat_t)
		if !ok || devStat == nil {
			continue
		}
		if uint64(devStat.Rdev) == targetDev {
			return entry.Name()
		}
	}
	return ""
}

func deviceMajorMinor(dev uint64) (uint32, uint32) {
	major := uint32((dev>>8)&0xfff) | uint32((dev>>32)&^uint64(0xfff))
	minor := uint32(dev&0xff) | uint32((dev>>12)&^uint64(0xff))
	return major, minor
}
