package monitor

// tu znajdują się pliki
const PATH string = "/sys/fs/cgroup/system.slice/slurmstepd.scope/"

type jobData struct {
	user string
	node string
}

type cpuMetrics struct {
	// RAM
	memoryCurrent uint64
	memoryAnon    uint64
	memoryPeak    uint64
	memoryLimit   uint64
	oomKills      uint64

	// CPU
	cpuUsageUsec  uint64
	cpuUserUsec   uint64
	cpuSystemUsec uint64
	throttledUsec uint64
}

// func initCPUMonitor() {

// 	JobCache := make(map[string]jobData)
// 	JobPid := make(map[string][]int32)
// 	JobMetrics := make(map[string]cpuMetrics)

// }
