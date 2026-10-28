package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	monitor "slurm-monitor/monitor"
)

const listenAddress = ":9840"

func main() {
	log.Printf("eden-job-exporter: start, listen at %s", listenAddress)

	monitor.StartMonitoring()

	http.Handle("/metrics", promhttp.Handler())

	err := http.ListenAndServe(listenAddress, nil)
	log.Fatal(err)
}
