package handlers

import (
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

const megabyteDiv uint64 = 1024 * 1024
const gigabyteDiv uint64 = megabyteDiv * 1024

// SystemInfo holds system information
type SystemInfo struct {
	OS           string
	Platform     string
	Hostname     string
	Procs        uint64
	TotalMem     uint64
	FreeMem      uint64
	UsedPercent  float64
}

// DiskInfo holds disk information
type DiskInfo struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// CPUInfo holds CPU information
type CPUInfo struct {
	ModelName    string
	Family       string
	Mhz          float64
	Percentages  []float64
}

// GetSystemInfo retrieves system information
func GetSystemInfo() (*SystemInfo, error) {
	runTimeOS := runtime.GOOS

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	hostStat, err := host.Info()
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		OS:          runTimeOS,
		Platform:    hostStat.Platform,
		Hostname:    hostStat.Hostname,
		Procs:       hostStat.Procs,
		TotalMem:    vmStat.Total / megabyteDiv,
		FreeMem:     vmStat.Free / megabyteDiv,
		UsedPercent: vmStat.UsedPercent,
	}, nil
}

// GetDiskInfo retrieves disk information
func GetDiskInfo() (*DiskInfo, error) {
	diskStat, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	return &DiskInfo{
		Total:       diskStat.Total / gigabyteDiv,
		Used:        diskStat.Used / gigabyteDiv,
		Free:        diskStat.Free / gigabyteDiv,
		UsedPercent: diskStat.UsedPercent,
	}, nil
}

// GetCPUInfo retrieves CPU information
func GetCPUInfo() (*CPUInfo, error) {
	cpuStat, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	percentage, err := cpu.Percent(0, true)
	if err != nil {
		return nil, err
	}

	var modelName, family string
	var mhz float64

	if len(cpuStat) > 0 {
		modelName = cpuStat[0].ModelName
		family = cpuStat[0].Family
		mhz = cpuStat[0].Mhz
	}

	return &CPUInfo{
		ModelName:   modelName,
		Family:      family,
		Mhz:         mhz,
		Percentages: percentage,
	}, nil
}
