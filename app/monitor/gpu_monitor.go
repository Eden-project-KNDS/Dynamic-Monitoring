package monitor

import (
	"bufio"
	"fmt"
	"log"
	"os"
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
}

type gpuMonitor struct {
	gpuCount int
	devices  []nvml.Device
	metrics  []gpuMetrics
}

func NewGPUMonitor() *gpuMonitor {
	ret := nvml.Init()

	if ret != nvml.SUCCESS {
		log.Printf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
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
			log.Printf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
			return &gpuMonitor{}
		}
		newGPUMonitor.devices[i] = device
	}
	return &newGPUMonitor
}

func parseProcCgroups(pid uint32) string {
	file, err := os.Open("/proc/" + strconv.FormatUint(uint64(pid), 10) + "/cgroup")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		splitLine := strings.Split(line, "/")

		for _, val := range splitLine {
			if len(val) > 3 && val[:3] == "job" {
				return val
			}
		}
	}
	return ""
}

// flag 0 print on terminal flag 1 send to prometheus
func (m *gpuMonitor) getGpuStats(flag int) *[]GPUStat {

	var gpuStats []GPUStat = make([]GPUStat, m.gpuCount)

	for i := 0; i < m.gpuCount; i++ {
		jobIdString := ""
		// reseting slice size without freeing memory
		util, retUtil := m.devices[i].GetUtilizationRates()
		// _v2 reports driver-reserved memory separately; v1 counts it as used (~450 MiB on an idle card).
		mem, retMem := m.devices[i].GetMemoryInfo_v2()
		temp, retTemp := m.devices[i].GetTemperature(nvml.TEMPERATURE_GPU)
		power, retPower := m.devices[i].GetPowerUsage() // Returns milliwatts
		proc, retPids := m.devices[i].GetComputeRunningProcesses()

		if retUtil == nvml.SUCCESS {
			m.metrics[i].coreUtilPercent = uint32(util.Gpu)
			m.metrics[i].memoryUtilPercent = uint32(util.Memory)
		}

		if retMem == nvml.SUCCESS {
			m.metrics[i].memoryUsedMB = mem.Used / (1024 * 1024)
			m.metrics[i].memoryTotalMB = mem.Total / (1024 * 1024)
		}

		if retTemp == nvml.SUCCESS {
			m.metrics[i].tempCelsius = uint32(temp)
		}

		if retPower == nvml.SUCCESS {
			m.metrics[i].powerWatts = float64(power) / 1000.0
		}
		if retPids == nvml.SUCCESS {
			for _, pid := range proc {
				jobIdString = parseProcCgroups(pid.Pid)
				fmt.Println(jobIdString)
				if jobIdString != "" {
					break
				}
			}
		}

		if flag == 0 {
			fmt.Printf("  Core Usage:   %d%%\n", m.metrics[i].coreUtilPercent)
			fmt.Printf("  VRAM Usage:   %d MB / %d MB (Mem Controller: %d%%)\n", m.metrics[i].memoryUsedMB, m.metrics[i].memoryTotalMB, m.metrics[i].memoryUtilPercent)
			fmt.Printf("  Temperature:  %d°C\n", m.metrics[i].tempCelsius)
			fmt.Printf("  Power Usage:  %.2f W\n\n", m.metrics[i].powerWatts)
		} else if flag == 1 {
			gpuStats[i].Index = i
			if uuid, ret := m.devices[i].GetUUID(); ret == nvml.SUCCESS {
				gpuStats[i].UUID = uuid
			}
			gpuStats[i].UtilPercent = m.metrics[i].coreUtilPercent
			gpuStats[i].JobID = jobIdString
			gpuStats[i].MemoryUsed = m.metrics[i].memoryUsedMB * 1024 * 1024 // For proper display in Grafana

		}
	}
	if flag == 0 {
		return nil
	} else if flag == 1 {
		return &gpuStats
	}
	return nil
}

func (m *gpuMonitor) GPUs() []GPUStat {
	return *(m.getGpuStats(1))
}

// Close releases NVML. Call it once, when the program shuts down.
func (m *gpuMonitor) Close() {
	if m.gpuCount == 0 {
		return
	}
	if ret := nvml.Shutdown(); ret != nvml.SUCCESS {
		log.Printf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
	}
}
