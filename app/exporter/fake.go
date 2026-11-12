package exporter

import (
	"time"

	monitor "slurm-monitor/monitor"
)

// FakeSource returns made-up data, so the exporter can be developed without Slurm or a GPU.
type FakeSource struct{}

// fakeStart lets the fake CPU counters grow over time, like real ones.
var fakeStart = time.Now()

func (FakeSource) GPUs() []monitor.GPUStat {
	return []monitor.GPUStat{
		{Index: 0, UUID: "GPU-fake-0", UtilPercent: 95, JobID: "1001", MemoryUsed: 30 << 30},
		{Index: 1, UUID: "GPU-fake-1", UtilPercent: 0, JobID: "1001", MemoryUsed: 0}, // allocated but idle
		{Index: 2, UUID: "GPU-fake-2", UtilPercent: 40, JobID: "1002", MemoryUsed: 10 << 30},
		{Index: 3, UUID: "GPU-fake-3", UtilPercent: 0, JobID: "", MemoryUsed: 0}, // free
	}
}

func (FakeSource) Jobs() []monitor.JobStat {
	elapsed := time.Since(fakeStart).Seconds()

	return []monitor.JobStat{
		{
			JobID: "1001", User: "alice", Account: "lab-a", Partition: "gpu", Name: "train",
			CPUUsageUsec:  uint64(elapsed * 3.5 * 1e6), // ~3.5 of 4 cores busy
			CPUsAllocated: 4,
			MemoryCurrent: 20 << 30,
			MemoryPeak:    24 << 30,
			MemoryLimit:   64 << 30,
		},
		{
			JobID: "1002", User: "bob", Account: "lab-b", Partition: "gpu", Name: "notebook",
			CPUUsageUsec:  uint64(elapsed * 0.5 * 1e6), // ~0.5 of 8 cores busy: wasted CPU
			CPUsAllocated: 8,
			MemoryCurrent: 2 << 30,
			MemoryPeak:    3 << 30,
			MemoryLimit:   0, // no limit
		},
	}
}
