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
	coreUtilPercent   uint8
	memoryUtilPercent uint8
	tempCelsius       uint8
	memoryUsedMB      uint64
	memoryTotalMB     uint64
	powerWatts        float64
}

type gpuMonitor struct {
	devices []nvml.Device
	metrics []gpuMetrics
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

	newGPUMonitor := gpuMonitor{make([]nvml.Device, gpuCount), make([]gpuMetrics, gpuCount)}

	for i := 0; i < gpuCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)

		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
		}
		newGPUMonitor.devices[i] = device
	}
	newGPUMonitor.start(gpuCount)
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
				for i := 0; i < gpuCount; i++ {
					util, retUtil := m.devices[i].GetUtilizationRates()
					mem, retMem := m.devices[i].GetMemoryInfo()
					temp, retTemp := m.devices[i].GetTemperature(nvml.TEMPERATURE_GPU)
					power, retPower := m.devices[i].GetPowerUsage() // Returns milliwatts

					if retUtil == nvml.SUCCESS {
						m.metrics[i].coreUtilPercent = uint8(util.Gpu)
						m.metrics[i].memoryUtilPercent = uint8(util.Memory)
					}

					if retMem == nvml.SUCCESS {
						m.metrics[i].memoryUsedMB = mem.Used / (1024 * 1024)
						m.metrics[i].memoryTotalMB = mem.Total / (1024 * 1024)
					}

					if retTemp == nvml.SUCCESS {
						m.metrics[i].tempCelsius = uint8(temp)
					}

					if retPower == nvml.SUCCESS {
						m.metrics[i].powerWatts = float64(power) / 1000.0
					}

					fmt.Printf("  Core Usage:   %d%%\n", m.metrics[i].coreUtilPercent)
					fmt.Printf("  VRAM Usage:   %d MB / %d MB (Mem Controller: %d%%)\n", m.metrics[i].memoryUsedMB, m.metrics[i].memoryTotalMB, m.metrics[i].memoryUtilPercent)
					fmt.Printf("  Temperature:  %d°C\n", m.metrics[i].tempCelsius)
					fmt.Printf("  Power Usage:  %.2f W\n\n", m.metrics[i].powerWatts)

				}

			}
		}
	}

}
