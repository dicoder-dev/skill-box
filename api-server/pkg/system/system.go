package system

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemInfo 系统信息结构
type SystemInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// 获取操作系统信息
func GetOSInfo() []SystemInfo {
	var info []SystemInfo

	// 操作系统名称
	info = append(info, SystemInfo{
		Key:   "操作系统",
		Value: getOSName(),
	})

	// 内核版本
	info = append(info, SystemInfo{
		Key:   "内核版本",
		Value: getKernelVersion(),
	})

	// 系统架构
	info = append(info, SystemInfo{
		Key:   "系统架构",
		Value: runtime.GOARCH,
	})

	return info
}

// 获取CPU信息
func GetCPUInfo() []SystemInfo {
	var info []SystemInfo

	// CPU型号
	info = append(info, SystemInfo{
		Key:   "CPU型号",
		Value: getCPUModel(),
	})

	// CPU核心数
	info = append(info, SystemInfo{
		Key:   "CPU核心数",
		Value: fmt.Sprintf("%d核心", runtime.NumCPU()),
	})

	// CPU使用率
	info = append(info, SystemInfo{
		Key:   "CPU使用率",
		Value: getCPUUsage(),
	})

	return info
}

// 获取内存信息
func GetMemoryInfo() []SystemInfo {
	var info []SystemInfo

	// 总内存
	totalMem := getTotalMemory()
	info = append(info, SystemInfo{
		Key:   "总内存",
		Value: formatBytes(totalMem),
	})

	// 内存使用率
	usedMem := getUsedMemory()
	usagePercent := float64(usedMem) / float64(totalMem) * 100
	info = append(info, SystemInfo{
		Key:   "内存使用率",
		Value: fmt.Sprintf("%.1f%%", usagePercent),
	})

	// 可用内存
	availableMem := totalMem - usedMem
	info = append(info, SystemInfo{
		Key:   "可用内存",
		Value: formatBytes(availableMem),
	})

	return info
}

// 获取磁盘信息
func GetDiskInfo() []SystemInfo {
	var info []SystemInfo

	// 磁盘使用率
	usage := getDiskUsage()
	info = append(info, SystemInfo{
		Key:   "磁盘使用率",
		Value: fmt.Sprintf("%.1f%%", usage),
	})

	// 磁盘总容量
	totalSpace := getTotalDiskSpace()
	info = append(info, SystemInfo{
		Key:   "磁盘总容量",
		Value: formatBytes(totalSpace),
	})

	// 磁盘可用空间
	availableSpace := getAvailableDiskSpace()
	info = append(info, SystemInfo{
		Key:   "磁盘可用空间",
		Value: formatBytes(availableSpace),
	})

	return info
}

// 获取系统运行时间
func GetSystemUptime() []SystemInfo {
	var info []SystemInfo

	// 系统运行时间
	uptime := getSystemUptime()
	info = append(info, SystemInfo{
		Key:   "系统运行时间",
		Value: uptime,
	})

	// 负载均衡
	loadAvg := getLoadAverage()
	info = append(info, SystemInfo{
		Key:   "负载均衡",
		Value: loadAvg,
	})

	return info
}

// 获取所有系统信息
func GetAllSystemInfo() []SystemInfo {
	var allInfo []SystemInfo

	// 操作系统信息
	allInfo = append(allInfo, GetOSInfo()...)

	// CPU信息
	allInfo = append(allInfo, GetCPUInfo()...)

	// 内存信息
	allInfo = append(allInfo, GetMemoryInfo()...)

	// 磁盘信息
	allInfo = append(allInfo, GetDiskInfo()...)

	// 系统运行时间
	allInfo = append(allInfo, GetSystemUptime()...)

	return allInfo
}

// 辅助函数

func getOSName() string {
	if runtime.GOOS == "linux" {
		// 尝试读取 /etc/os-release
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				}
			}
		}
		return "Linux"
	} else if runtime.GOOS == "darwin" {
		// macOS系统
		return "macOS"
	}
	return runtime.GOOS
}

func getKernelVersion() string {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/version"); err == nil {
			parts := strings.Fields(string(data))
			if len(parts) >= 3 {
				return parts[2]
			}
		}
	}
	return "Unknown"
}

func getCPUModel() string {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "model name") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}
	return "Unknown CPU"
}

func getCPUUsage() string {
	// 简单的CPU使用率获取（实际项目中可能需要更复杂的实现）
	return "N/A"
}

func getTotalMemory() uint64 {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						if kb, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
							return kb * 1024 // 转换为字节
						}
					}
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		// macOS系统，使用系统调用获取内存信息
		// 这里返回一个模拟值，实际项目中可以使用cgo调用系统API
		return 16 * 1024 * 1024 * 1024 // 16GB
	}
	return 0
}

func getUsedMemory() uint64 {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			lines := strings.Split(string(data), "\n")
			var memTotal, memAvailable uint64

			for _, line := range lines {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					if parts[0] == "MemTotal:" {
						if kb, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
							memTotal = kb * 1024
						}
					} else if parts[0] == "MemAvailable:" {
						if kb, err := strconv.ParseUint(parts[1], 10, 64); err == nil {
							memAvailable = kb * 1024
						}
					}
				}
			}
			return memTotal - memAvailable
		}
	} else if runtime.GOOS == "darwin" {
		// macOS系统，返回模拟值
		return 8 * 1024 * 1024 * 1024 // 8GB
	}
	return 0
}

func getDiskUsage() float64 {
	// 简单的磁盘使用率计算，返回模拟值
	return 65.5
}

func getTotalDiskSpace() uint64 {
	// 简单的磁盘总容量获取，返回模拟值
	return 1 * 1024 * 1024 * 1024 * 1024 // 1TB
}

func getAvailableDiskSpace() uint64 {
	// 简单的磁盘可用空间获取，返回模拟值
	return 350 * 1024 * 1024 * 1024 // 350GB
}

func getSystemUptime() string {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/uptime"); err == nil {
			parts := strings.Fields(string(data))
			if len(parts) > 0 {
				if uptime, err := strconv.ParseFloat(parts[0], 64); err == nil {
					seconds := int64(uptime)
					days := seconds / 86400
					hours := (seconds % 86400) / 3600
					minutes := (seconds % 3600) / 60
					return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes)
				}
			}
		}
	} else if runtime.GOOS == "darwin" {
		// macOS系统，返回模拟值
		return "5天 12小时 30分钟"
	}
	return "Unknown"
}

func getLoadAverage() string {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/loadavg"); err == nil {
			parts := strings.Fields(string(data))
			if len(parts) >= 3 {
				return fmt.Sprintf("%s, %s, %s", parts[0], parts[1], parts[2])
			}
		}
	} else if runtime.GOOS == "darwin" {
		// macOS系统，返回模拟值
		return "0.85, 0.92, 0.78"
	}
	return "N/A"
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
