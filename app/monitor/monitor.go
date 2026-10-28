package monitor

func StartMonitoring() {
	go initGPUMonitoring()
	go initCPUMonitor()

}
