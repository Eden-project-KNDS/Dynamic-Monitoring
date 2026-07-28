package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"strings"
)

func openFiles(slurmPID *string) (*os.File, *os.File, error) {
	var cpuFileName string = "usage_cpu_ram_" + *slurmPID + ".log"
	var gpuFileName string = "usage_gpu_" + *slurmPID + ".log"

	cpuFile, err := os.Open(cpuFileName)
	if err != nil {
		log.Fatalf("couldn't open cpu log file %v", err)
		return nil, nil, err
	}

	gpuFile, err := os.Open(gpuFileName)
	if err != nil {
		log.Fatalf("couldn't open gpu log file %v", err)
		return nil, nil, err
	}
	return cpuFile, gpuFile, nil

}

func parseLine(cpuLine *string, gpuLine *string) (*DBEntry, error) {
	dbRow := &DBEntry{}

	cpuLineSpilt := strings.Fields(*cpuLine)

	for i :=2; i < cpuLineSpilt



}

func readFilesSaveToDb(accountName *string, slurmPID *string, tx *sql.Tx, stmt *sql.Stmt) error {

	cpuFile, gpuFile, err := openFiles(slurmPID)
	if err != nil {
		return err
	}
	defer cpuFile.Close()
	defer gpuFile.Close()

	cpuScanner := bufio.NewScanner(cpuFile)
	gpuScanner := bufio.NewScanner(gpuFile)

	// SKIP THE HEADERS ---

	// Skip 2 header lines in the CPU log
	cpuScanner.Scan()
	cpuScanner.Scan()

	// Skip 1 header line in the GPU log
	gpuScanner.Scan()

	for cpuScanner.Scan() && gpuScanner.Scan() {
		cpuLine := cpuScanner.Text()
		gpuLine := gpuScanner.Text()

		dbRow, err := parseLine(&cpuLine, &gpuLine)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(dbRow.Time, dbRow.JobId, dbRow.Account, dbRow.PID, dbRow.UsrPercentage,
			dbRow.SystemPercentage, dbRow.GuestPercentage, dbRow.WaitPercentage, dbRow.CpuPercentage,
			dbRow.Cpu, dbRow.MinfltsPerS, dbRow.MajfltsPerS, dbRow.VSZ, dbRow.RSS, dbRow.RamPercentage,
			dbRow.UtilizationGpuPercentage, dbRow.UtilizationGpuMemory, dbRow.MemoryGpuUsedMib)

		if err != nil {
			tx.Rollback()
			log.Fatalf("Error when buffering metrics: %v", err)
			return err
		}

	}

	return nil
}
