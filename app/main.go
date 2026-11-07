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

func usage(program string) {
	fmt.Printf("Program %s takes 2 arg, node name and port\n  How to run: ./%s node_name port", program, program)

}

func main() {

	if len(os.Args) != 3 {
		usage(os.Args[0])
		return
	}

	node := os.Args[1]
	listenAddress := ":" + os.Args[2]

	// s is shared by all goroutines, so they stop together on SIGINT or SIGTERM.
	s, stop := synchronization.CreateSignalHandling()
	defer stop()

	fmt.Printf("Start\n")
	gpus := monitor.NewGPUMonitor()
	jobs := monitor.NewAllJobs(node)

	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter.NewCollector(gpus, jobs))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{MaxRequestsInFlight: 1}))

	srv := &http.Server{Addr: listenAddress}

	// Shuts the http server down on a shutdown signal; open requests get up to 5 s to finish.
	go func() {
		defer s.Wg.Done()
		<-s.Ctx.Done()
		fmt.Println("\nStopping monitor")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down http server: %v", err)
		}
	}()

	// On a server error (e.g. port in use) stop the program the same way as on a signal.
	serverErr := srv.ListenAndServe()
	if serverErr != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", serverErr)
		stop()
	}

	s.Wg.Wait()
	gpus.Close()
	fmt.Println("Monitor stopped")

	if serverErr != http.ErrServerClosed {
		os.Exit(1)
	}
}
