//go:build windows

package runner

// readDiskUsageSyscall returns 0 on Windows (not supported in runner mode).
func readDiskUsageSyscall(_ string) (float64, error) {
	return 0.0, nil
}
