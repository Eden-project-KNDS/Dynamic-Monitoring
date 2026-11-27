package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"slurm-monitor/exporter"
	helper "slurm-monitor/helper"
	monitor "slurm-monitor/monitor"

	"slurm-monitor/synchronization"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func usage(program string) {
	fmt.Printf("Program %s takes 1 arg, port\n  How to run: ./%s port", program, program)
}

func run(listenAddress string) {

	// s is shared by all goroutines, so they stop together on SIGINT or SIGTERM.
	s, stop := synchronization.CreateSignalHandling()
	defer stop()

	helper.GeneralLogger.Printf("Start\n")

	gpus := monitor.NewGPUMonitor()
	jobs := monitor.NewAllJobs()

	reg := prometheus.NewRegistry()
	reg.MustRegister(exporter.NewCollector(gpus, jobs))

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{MaxRequestsInFlight: 1}))

	srv := &http.Server{Addr: listenAddress}

	// Shuts the http server down on a shutdown signal; open requests get up to 5 s to finish.
	go func() {
		defer s.Wg.Done()
		<-s.Ctx.Done()
		helper.GeneralLogger.Println("\nStopping monitor")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			helper.ErrorLogger.Printf("Error shutting down http server: %v", err)
		}
	}()

	// On a server error (e.g. port in use) stop the program the same way as on a signal.
	serverErr := srv.ListenAndServe()
	if serverErr != http.ErrServerClosed {
		helper.ErrorLogger.Printf("HTTP server error: %v", serverErr)
		stop()
	}

	s.Wg.Wait()
	gpus.Close()
	fmt.Println("Monitor stopped")

	if serverErr != http.ErrServerClosed {
		os.Exit(1)
	}
}

func main() {

	if len(os.Args) != 2 {
		usage(os.Args[0])
		return
	}
	// Log to a file. If that fails, keep logging to stderr
	if err := helper.InitLogger(); err != nil {
		helper.ErrorLogger.Printf("Logging to stderr instead of a file: %v", err)
	}

	// Without scontrol the monitor still serves GPU metrics. Job metrics stay empty.
	scontrolPath, err := helper.FindScontrol()
	if err != nil {
		helper.ErrorLogger.Printf("Job metrics disabled: %v", err)
	} else {
		helper.GeneralLogger.Printf("Using scontrol at %s", scontrolPath)
	}
	monitor.ScontrolPath = scontrolPath

	listenAddress := ":" + os.Args[1]
	run(listenAddress)
}
