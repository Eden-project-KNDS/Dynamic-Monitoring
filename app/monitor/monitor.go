package monitor

const TimeInterval int = 5

func StartMonitoring(nodeName string) {
	//initGPUMonitoring(nodeName)
	newAllJobs().run(nodeName)

}
