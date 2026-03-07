// Package runner implements the GitLab Runner Agent lifecycle manager.
package runner

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// RunnerMetrics holds a point-in-time snapshot of runner health data.
// These values are reported to MotherGoose and used by UglyFox for pruning decisions.
type RunnerMetrics struct {
	RunnerID         string    `json:"runner_id"`
	EggName          string    `json:"egg_name"`
	State            string    `json:"state"`
	JobCount         int       `json:"job_count"`
	LastHeartbeat    time.Time `json:"last_heartbeat"`
	CPUUsage         float64   `json:"cpu_usage"`
	MemoryUsage      float64   `json:"memory_usage"`
	DiskUsage        float64   `json:"disk_usage"`
	AgentVersion     string    `json:"agent_version"`
	FailureCount     int       `json:"failure_count"`
	LastJobTimestamp time.Time `json:"last_job_timestamp"`
}

// SystemStats holds raw OS-level resource readings.
type SystemStats struct {
	CPUUsagePercent    float64
	MemoryUsagePercent float64
	DiskUsagePercent   float64
}

// StatReader is the interface for reading system resource statistics.
// Using an interface allows tests to inject deterministic values.
type StatReader interface {
	ReadStats() (SystemStats, error)
}

// OSStatReader reads real system statistics from the OS.
type OSStatReader struct{}

// ReadStats collects CPU, memory, and disk usage from the OS.
// It uses /proc on Linux and falls back to safe defaults on other platforms.
func (r *OSStatReader) ReadStats() (SystemStats, error) {
	cpu, err := readCPUUsage()
	if err != nil {
		cpu = 0.0
	}

	mem, err := readMemoryUsage()
	if err != nil {
		mem = 0.0
	}

	disk, err := readDiskUsage("/")
	if err != nil {
		disk = 0.0
	}

	return SystemStats{
		CPUUsagePercent:    cpu,
		MemoryUsagePercent: mem,
		DiskUsagePercent:   disk,
	}, nil
}

// readCPUUsage reads CPU usage from /proc/stat on Linux.
// Returns a value in [0, 100]. Returns 0 on non-Linux platforms.
func readCPUUsage() (float64, error) {
	if runtime.GOOS != "linux" {
		return 0.0, nil
	}

	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0.0, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	// Parse the first "cpu" line: cpu user nice system idle iowait irq softirq steal guest guest_nice
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0.0, fmt.Errorf("unexpected /proc/stat format")
		}

		var vals [10]uint64
		for i := 1; i < len(fields) && i <= 10; i++ {
			v, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return 0.0, fmt.Errorf("failed to parse /proc/stat field %d: %w", i, err)
			}
			vals[i-1] = v
		}

		idle := vals[3] + vals[4] // idle + iowait
		total := uint64(0)
		for _, v := range vals {
			total += v
		}
		if total == 0 {
			return 0.0, nil
		}
		return (1.0 - float64(idle)/float64(total)) * 100.0, nil
	}

	return 0.0, fmt.Errorf("cpu line not found in /proc/stat")
}

// readMemoryUsage reads memory usage from /proc/meminfo on Linux.
// Returns a value in [0, 100]. Returns 0 on non-Linux platforms.
func readMemoryUsage() (float64, error) {
	if runtime.GOOS != "linux" {
		return 0.0, nil
	}

	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0.0, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}

	var memTotal, memAvailable uint64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		}
	}

	if memTotal == 0 {
		return 0.0, fmt.Errorf("MemTotal not found in /proc/meminfo")
	}

	used := memTotal - memAvailable
	return float64(used) / float64(memTotal) * 100.0, nil
}

// readDiskUsage reads disk usage for the given path using syscall.Statfs.
// Returns a value in [0, 100].
func readDiskUsage(path string) (float64, error) {
	return readDiskUsageSyscall(path)
}
