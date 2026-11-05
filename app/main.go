package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"slurm-monitor/exporter"
	monitor "slurm-monitor/monitor"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func usage(program string) {
	fmt.Printf("Program %s takes 2 arg, node name and port\n  How to run: ./%s node_name port", program, program)

}

func main() {

	if len(os.Args) != 3 {
		usage(os.Args[0])
		return
	}

	node := os.Args[1]
	listenAddress := os.Args[2]

	fmt.Printf("Start\n")
	gpus := monitor.NewGPUMonitor()
	jobs := monitor.StartMonitoring(node)

	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter.NewCollector(gpus, jobs))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	err := http.ListenAndServe(listenAddress, nil)
	log.Fatal(err)
}
