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

	fake := exporter.FakeSource{}
	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter.NewCollector(fake, fake))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	err := http.ListenAndServe(listenAddress, nil)
	log.Fatal(err)
}
