//go:build linux || darwin

package runner

import (
	"fmt"
	"syscall"
)

// readDiskUsageSyscall returns disk usage percentage for the given path on Unix.
func readDiskUsageSyscall(path string) (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0.0, fmt.Errorf("statfs(%q) failed: %w", path, err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total == 0 {
		return 0.0, nil
	}

	used := total - free
	return float64(used) / float64(total) * 100.0, nil
}
