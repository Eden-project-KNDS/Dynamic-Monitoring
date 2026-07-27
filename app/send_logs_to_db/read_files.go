package main

import (
	"log"
	"os"
)

func readFilesSaveToDb(accountName *string, slurmPID *string) error {
	var cpuFileName string = "usage_cpu_ram_" + *slurmPID + ".log"
	var gpuFileName string = "usage_gpu_" + *slurmPID + ".log"

	cpuFile, err := os.Open(cpuFileName)
	if err != nil {
		log.Fatalf("couldn't open cpu log file %v", err)
		return err
	}
	defer cpuFile.Close()

	gpuFile, err := os.Open(gpuFileName)
	if err != nil {
		log.Fatalf("couldn't open gpu log file %v", err)
		return err
	}
	defer gpuFile.Close()

	return nil
}
