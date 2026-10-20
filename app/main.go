package main

import (
	"fmt"
	monitor "slurm-monitor/monitor"
)

func main() {
	fmt.Printf("Start\n")
	monitor.StartMonitoring()

}
