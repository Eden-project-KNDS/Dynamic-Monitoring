package monitor

import "slurm-monitor/synchronization"

const TimeInterval int = 5

// StartMonitoring starts the job discovery loop in the background
// and returns the job source for the exporter. The loop stops when s.Ctx is cancelled.
func StartMonitoring(nodeName string, s *synchronization.SyncStruct) *allJobs {
	jobs := newAllJobs()
	go jobs.run(nodeName, s)
	return jobs
}
