package main

import (
	"bufio"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
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
	gpuLineSplit := strings.Fields(*gpuLine)

	var err error = nil

	dbRow.PID = cpuLineSpilt[3]
	dbRow.UsrPercentage, err = strconv.ParseFloat(cpuLineSpilt[4], 64)
	dbRow.SystemPercentage, err = strconv.ParseFloat(cpuLineSpilt[5], 64)
	dbRow.GuestPercentage, err = strconv.ParseFloat(cpuLineSpilt[6], 64)
	dbRow.WaitPercentage, err = strconv.ParseFloat(cpuLineSpilt[7], 64)
	dbRow.CpuPercentage, err = strconv.ParseFloat(cpuLineSpilt[8], 64)
	dbRow.Cpu, err = strconv.ParseFloat(cpuLineSpilt[9], 64)
	dbRow.MinfltsPerS, err = strconv.ParseFloat(cpuLineSpilt[10], 64)
	dbRow.MajfltsPerS, err = strconv.ParseFloat(cpuLineSpilt[11], 64)
	dbRow.VSZ, err = strconv.ParseFloat(cpuLineSpilt[12], 64)
	dbRow.RSS, err = strconv.ParseFloat(cpuLineSpilt[13], 64)
	dbRow.RamPercentage, err = strconv.ParseFloat(cpuLineSpilt[14], 64)

	dbRow.UtilizationGpuPercentage, err = strconv.ParseFloat(gpuLineSplit[2], 64)
	dbRow.UtilizationGpuMemory, err = strconv.ParseFloat(gpuLineSplit[3], 64)
	dbRow.MemoryGpuUsedMib, err = strconv.ParseFloat(gpuLineSplit[4], 64)

	layout := "2006/01/02 15:04:05.000"
	combinedString := gpuLineSplit[0] + " " + gpuLineSplit[1]

	parsedTime, err := time.Parse(layout, combinedString)
	if err != nil {
		log.Fatalf("Error parsing time %v ", err)
		return nil, err
	}
	dbRow.Time = parsedTime

	return dbRow, nil

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
		dbRow.Account = *accountName
		dbRow.JobId = *slurmPID

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
