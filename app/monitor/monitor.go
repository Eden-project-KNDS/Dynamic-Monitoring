package monitor

const TimeInterval int = 5

// StartMonitoring starts the job discovery loop in the background
// and returns the job source for the exporter.
func StartMonitoring(nodeName string) *allJobs {
	jobs := newAllJobs()
	go jobs.run(nodeName)
	return jobs
}
