package monitor

const TimeInterval int = 5

func StartMonitoring() {
	go initGPUMonitoring()
	go initCPUMonitor()

}
