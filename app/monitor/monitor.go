package monitor

const TimeInterval int = 5

func StartMonitoring(nodeName string) {
	//go initGPUMonitoring(nodeName)
	initCPUMonitor(nodeName)

}
