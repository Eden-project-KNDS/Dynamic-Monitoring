package main

import (
	"fmt"
	"os"
	monitor "slurm-monitor/monitor"
)

func usage(program string) {
	fmt.Printf("Program %s takes 1 arg, node name\n  How to run: ./%s node_name", program, program)

}

func main() {

	if len(os.Args) != 2 {
		usage(os.Args[0])
		return
	}

	node := os.Args[1]

	fmt.Printf("Start\n")
	monitor.StartMonitoring(node)

}
