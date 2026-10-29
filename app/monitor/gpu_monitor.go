package monitor

import (
	"fmt"
	"log"
	"time"

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

func initGPUMonitoring(nodeName string) {
	ret := nvml.Init()

	if ret != nvml.SUCCESS {
		log.Printf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
		return
	}

	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()

	gpuCount := getGPUCount()
	if gpuCount == 0 {
		return
	}

	newGPUMonitor := gpuMonitor{gpuCount, make([]nvml.Device, gpuCount), make([]gpuMetrics, gpuCount)}

	for i := 0; i < gpuCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)

		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
		}
		newGPUMonitor.devices[i] = device
	}
	newGPUMonitor.start(gpuCount)
}

// flag 0 print on terminal flag 1 send to prometheus
func (m gpuMonitor) getGpuStats(flag int) []GPUStat {

	var gpuStats []GPUStat = make([]GPUStat, m.gpuCount)

	for i := 0; i < m.gpuCount; i++ {
		util, retUtil := m.devices[i].GetUtilizationRates()
		mem, retMem := m.devices[i].GetMemoryInfo()
		temp, retTemp := m.devices[i].GetTemperature(nvml.TEMPERATURE_GPU)
		power, retPower := m.devices[i].GetPowerUsage() // Returns milliwatts

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

		if flag == 0 {
			fmt.Printf("  Core Usage:   %d%%\n", m.metrics[i].coreUtilPercent)
			fmt.Printf("  VRAM Usage:   %d MB / %d MB (Mem Controller: %d%%)\n", m.metrics[i].memoryUsedMB, m.metrics[i].memoryTotalMB, m.metrics[i].memoryUtilPercent)
			fmt.Printf("  Temperature:  %d°C\n", m.metrics[i].tempCelsius)
			fmt.Printf("  Power Usage:  %.2f W\n\n", m.metrics[i].powerWatts)
		} else if flag == 1 {
			gpuStats[i].Index = i
			gpuStats[i].UUID = "dummy"
			gpuStats[i].UtilPercent = m.metrics[i].coreUtilPercent
			gpuStats[i].JobID = "dummy"
			gpuStats[i].MemoryUsed = m.metrics[i].memoryUsedMB

		}
	}
	if flag == 0 {
		return nil
	} else if flag == 1 {
		return gpuStats
	}
	return nil
}

func (m gpuMonitor) GPUs() []GPUStat {
	return m.getGpuStats(1)
}

func (m gpuMonitor) start(gpuCount int) {

	ticker := time.NewTicker(time.Duration(TimeInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {

		//case obsługa sygnału wyjścia

		case t := <-ticker.C:
			{
				fmt.Printf("%s\n", t.Format("2006-01-02 15:04:05"))
				m.getGpuStats(0)
			}
		}
	}

}
