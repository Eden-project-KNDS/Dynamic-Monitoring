package exporter

import monitor "slurm-monitor/monitor"

// FakeSource return fixed data
type FakeSource struct{}

func (FakeSource) GPUs() []monitor.GPUStat {
	return []monitor.GPUStat{
		{Index: 0, UUID: "GPU-fake-0", UtilPercent: 95, JobID: "1001"},
		{Index: 1, UUID: "GPU-fake-1", UtilPercent: 0, JobID: "1001"}, // allocated but idle
		{Index: 2, UUID: "GPU-fake-2", UtilPercent: 40, JobID: "1002"},
		{Index: 3, UUID: "GPU-fake-3", UtilPercent: 0, JobID: ""}, // free
	}
}
