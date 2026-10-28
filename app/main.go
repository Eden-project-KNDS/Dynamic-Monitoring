package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"slurm-monitor/exporter"
	monitor "slurm-monitor/monitor"
)

const listenAddress = ":9840"

func main() {
	log.Printf("eden-job-exporter: start, listen at %s", listenAddress)

	monitor.StartMonitoring()

	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter.NewCollector(exporter.FakeSource{}))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	err := http.ListenAndServe(listenAddress, nil)
	log.Fatal(err)
}
