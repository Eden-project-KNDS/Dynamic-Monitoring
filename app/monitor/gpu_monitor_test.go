package monitor

import (
	"testing"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
)

func checkHardware(t *testing.T) {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		t.Skipf("No nvidia devices: %v", nvml.ErrorString(ret))
	}
	nvml.Shutdown()
}

func TestNewGPUMonitor(t *testing.T) {
	checkHardware(t)
	monitor := NewGPUMonitor()
	if monitor == nil {
		t.Fatal("monitor == nil\n")
	}
	defer monitor.Close()

	if monitor.gpuCount < 0 {
		t.Errorf("Gpu count can't be a negative number : %v\n", monitor.gpuCount)
	}
	if monitor.gpuCount > 0 && len(monitor.devices) != monitor.gpuCount {
		t.Errorf("Expected %d devices, got %d\n", monitor.gpuCount, len(monitor.devices))
	}
}

func TestGpuGetStats(t *testing.T) {
	checkHardware(t)
	monitor := NewGPUMonitor()
	defer monitor.Close()

	if monitor.gpuCount < 0 {
		t.Skip("No gpu to test\n")
	}
	stats := monitor.getGpuStats(1)

	if len(stats) != monitor.gpuCount {
		t.Errorf("Expected %d stats, recieved %d", monitor.gpuCount, len(stats))
	}

	if len(stats) > 0 {
		if stats[0].UUID == "" {
			t.Error("Expected UUID gpu, got empty string")
		}
	}

}
