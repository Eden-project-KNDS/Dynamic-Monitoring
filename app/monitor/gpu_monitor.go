package monitor

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	helper "slurm-monitor/helper"
	"strconv"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

// fix change to ENV

func getGPUCount() int {
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		log.Printf("Unable to get device count: %v", nvml.ErrorString(ret))
		return 0
	}

	return count
}

type gpuMetrics struct {
	coreUtilPercent   uint32
	memoryUtilPercent uint32
	tempCelsius       uint32
	memoryUsedMB      uint64
	memoryTotalMB     uint64
	powerWatts        float64
	jobId             string
}

type gpuMonitor struct {
	gpuCount int
	devices  []nvml.Device
	metrics  []gpuMetrics
}

func NewGPUMonitor() *gpuMonitor {
	ret := nvml.Init()

	if ret != nvml.SUCCESS {
		helper.ErrorLogger.Printf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
		return &gpuMonitor{}
	}

	gpuCount := getGPUCount()
	if gpuCount == 0 {
		return &gpuMonitor{}
	}

	newGPUMonitor := gpuMonitor{gpuCount, make([]nvml.Device, gpuCount), make([]gpuMetrics, gpuCount)}

	for i := 0; i < gpuCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)

		if ret != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
			return &gpuMonitor{}
		}
		newGPUMonitor.devices[i] = device
	}
	return &newGPUMonitor
}

func tryParseProcCgroups(pid uint32) (string, error) {
	file, err := os.Open("/proc/" + strconv.FormatUint(uint64(pid), 10) + "/cgroup")
	if err != nil {
		return "", fmt.Errorf("Couldn't open cgroup for PID %d: %w", pid, err)
	}
	defer file.Close()

	return tryParseCgroupReader(file)
}

func tryParseCgroupReader(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		splitLine := strings.Split(line, "/")

		for _, val := range splitLine {
			if strings.HasPrefix(val, PATTERN) {
				jobId := strings.TrimPrefix(val, PATTERN) // job_{number} -> {number}
				id, err := strconv.ParseUint(jobId, 10, 64)
				if err != nil {
					return "", fmt.Errorf("Invalid job ID %q: %w", jobId, err)
				}
				if id == 0 {
					return "", fmt.Errorf("Job ID must be positive: %q", jobId)
				}
				return jobId, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("Scanning finished with error %w", err)
	}
	return "", nil
}

func (m *gpuMonitor) getGpuStats() []GPUStat {
	var gpuStats []GPUStat = make([]GPUStat, 0, m.gpuCount)

	for i := 0; i < m.gpuCount; i++ {
		m.metrics[i] = gpuMetrics{}
		uuid, retUUID := m.devices[i].GetUUID()
		if retUUID != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Couldn't read UUID for device index %d: %v", i, nvml.ErrorString(retUUID))
			continue
		}
		if uuid == "" {
			helper.ErrorLogger.Printf("Empty UUID for device index %d", i)
			continue
		}
		util, retUtil := m.devices[i].GetUtilizationRates()
		// _v2 reports driver-reserved memory separately; v1 counts it as used (~450 MiB on an idle card).
		mem, retMem := m.devices[i].GetMemoryInfo_v2()
		temp, retTemp := m.devices[i].GetTemperature(nvml.TEMPERATURE_GPU)
		power, retPower := m.devices[i].GetPowerUsage() // Returns milliwatts
		proc, retPids := m.devices[i].GetComputeRunningProcesses()
		if retUtil != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Couldn't read utilization for device index %d: %v", i, nvml.ErrorString(retUtil))
		} else {
			m.metrics[i].coreUtilPercent = uint32(util.Gpu)
			m.metrics[i].memoryUtilPercent = uint32(util.Memory)
		}
		if retMem != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Couldn't read memory info for device index %d: %v", i, nvml.ErrorString(retMem))
		} else {
			m.metrics[i].memoryUsedMB = mem.Used / (1024 * 1024)
			m.metrics[i].memoryTotalMB = mem.Total / (1024 * 1024)
		}
		if retTemp != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Couldn't read temperature for device index %d: %v", i, nvml.ErrorString(retTemp))
		} else {
			m.metrics[i].tempCelsius = uint32(temp)
		}
		if retPower != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Couldn't read power usage for device index %d: %v", i, nvml.ErrorString(retPower))
		} else {
			m.metrics[i].powerWatts = float64(power) / 1000.0
		}
		if retPids != nvml.SUCCESS {
			helper.ErrorLogger.Printf("Couldn't read PIDs for device index %d: %v", i, nvml.ErrorString(retPids))
			proc = nil
		}

		for _, pid := range proc {
			jobIdString, err := tryParseProcCgroups(pid.Pid)
			if err != nil {
				helper.ErrorLogger.Printf("Couldn't read job for device index %d, PID %d: %v", i, pid.Pid, err)
				continue
			}
			if jobIdString != "" {
				m.metrics[i].jobId = jobIdString
				break
			}
		}

		gpuStats = append(gpuStats, GPUStat{
			Index:            i,
			UUID:             uuid,
			UtilPercent:      m.metrics[i].coreUtilPercent,
			UtilPercentValid: retUtil == nvml.SUCCESS,
			JobID:            m.metrics[i].jobId,
			MemoryUsed:       m.metrics[i].memoryUsedMB * 1024 * 1024, // For proper display in Grafana
			MemoryUsedValid:  retMem == nvml.SUCCESS,
		})
	}
	return gpuStats
}

func (m *gpuMonitor) GPUs() []GPUStat {
	return m.getGpuStats()
}

// Close releases NVML. Call it once, when the program shuts down.
func (m *gpuMonitor) Close() {
	if m.gpuCount == 0 {
		return
	}
	if ret := nvml.Shutdown(); ret != nvml.SUCCESS {
		helper.ErrorLogger.Printf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
	}
}
