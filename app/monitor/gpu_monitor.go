package monitor

import (
	"log"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func getGPUCount() int {

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		log.Fatalf("Unable to get device count: %v", nvml.ErrorString(ret))
		return 0
	}

	return count

}

type GPUMonitor struct {
	devices []nvml.Device
}

func initGPUMonitoring() {
	ret := nvml.Init()

	if ret != nvml.SUCCESS {
		log.Fatalf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
		return
	}

	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()

	gpuCount := getGPUCount()

	newGPUMonitor := GPUMonitor{make([]nvml.Device, gpuCount)}

	for i := 0; i < gpuCount; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)

		if ret != nvml.SUCCESS {
			log.Fatalf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
		}
		newGPUMonitor.devices[i] = device
	}

}
