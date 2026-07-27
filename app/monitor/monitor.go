package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func main() {

	cudaPID := flag.String("job-id", "", "PID of the process to monitor")
	programName := flag.String("name", "", "name of the proccess to monitor")
	flag.Parse()

	if *cudaPID == "" {
		log.Fatal("CUDA PID not found")
		return
	}
	if *programName == "" {
		log.Fatal("Program name not found")
		return
	}
	log.Printf("Uruchamiam eden-monitor dla Job ID: %s (Monitor PID: %d)", *cudaPID, os.Getpid())

	usage_cpu_ram, err := os.Create("usage_cpu_ram.log")
	if err != nil {
		log.Fatal("Error creating file")
		return
	}
	defer usage_cpu_ram.Close()

	usage_gpu, err := os.Create("usage_gpu.log")
	if err != nil {
		log.Fatal("Error creating file")
		return
	}
	defer usage_gpu.Close()

	cpu_ram_cmd := exec.Command("pidstat", "-h", "-r", "-u", "-t", "-C", *programName, "1")
	cpu_ram_cmd.Stdout = usage_cpu_ram
	cpu_ram_cmd.Stderr = os.Stderr

	err = cpu_ram_cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start pidstat: %v", err)
	}

	gpu_command := exec.Command("nvidia-smi", "--query-gpu=timestamp,utilization.gpu,utilization.memory,memory.used", "--format=csv", "-l", "1")
	gpu_command.Stdout = usage_gpu
	gpu_command.Stderr = os.Stderr

	err1 := gpu_command.Start()
	if err1 != nil {
		log.Fatalf("Failed to start nvidia smi: %v", err1)
	}

	//Signal Handling

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs

	log.Printf("Receive signal %v, Starting cleaning...", sig)

	if cpu_ram_cmd.Process != nil {
		_ = cpu_ram_cmd.Process.Kill()
		log.Print("Stopped ram and cpu monitoring")
	}

	if gpu_command.Process != nil {
		_ = gpu_command.Process.Kill()
		log.Print("Stopped gpu monitoring")
	}

}
