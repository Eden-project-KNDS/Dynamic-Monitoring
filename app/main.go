package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slurm-monitor/exporter"
	monitor "slurm-monitor/monitor"
	"slurm-monitor/synchronization"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const listenAddress = ":9840"

func usage(program string) {
	fmt.Printf("Program %s takes 1 arg, node name\n  How to run: ./%s node_name", program, program)

}

func main() {

	if len(os.Args) != 2 {
		usage(os.Args[0])
		return
	}

	node := os.Args[1]

	// s is shared by all goroutines, so they stop together on SIGINT or SIGTERM.
	s, stop := synchronization.CreateSignalHandling()
	defer stop()

	fmt.Printf("Start\n")
	gpus := monitor.NewGPUMonitor()
	jobs := monitor.StartMonitoring(node, s)

	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter.NewCollector(gpus, jobs))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	// http.Server instead of http.ListenAndServe, because only a server can be shut down.
	srv := &http.Server{Addr: listenAddress}

	// Shuts the http server down on a shutdown signal; open requests get up to 5 s to finish.
	go func() {
		defer s.Wg.Done()
		<-s.Ctx.Done()
		fmt.Println("Shutdown signal received, stopping")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down http server: %v", err)
		}
	}()

	// ListenAndServe returns ErrServerClosed right after Shutdown is called.
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}

	// Wait until all goroutines have stopped.
	s.Wg.Wait()
	fmt.Println("Monitor stopped")
}
