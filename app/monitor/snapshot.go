package monitor

// GPU is the state of a single GPU card.
type GPUStat struct {
	Index       int
	UUID        string
	UtilPercent uint32 // 0-100
	JobID       string // "" = card not allocated to any job
	MemoryUsed  uint64
}

// JobStat is the state of a single Slurm job on this node.
type JobStat struct {
	// Job description (from Slurm).
	JobID     string
	User      string
	Account   string
	Partition string
	Name      string

	// CPU (from cgroup).
	CPUUsageUsec  uint64 // cpu.stat usage_usec, only grows
	CPUsAllocated int

	// Memory (from cgroup), in bytes.
	MemoryCurrent uint64
	MemoryPeak    uint64
	MemoryLimit   uint64 // memory.max 0 = no limit
}
