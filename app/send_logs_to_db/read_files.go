package main

import (
	"bufio"
	"database/sql"
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func openFile(slurmPID *string, metric logType) (*os.File, error) {

	var fileName string
	if metric == CPUMetric {
		fileName = "usage_cpu_ram_" + *slurmPID + ".log"
	} else if metric == GPUMetric {
		fileName = "usage_gpu_" + *slurmPID + ".log"
	}

	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("couldn't open %v log file %v", metric, err)
		return nil, err
	}
	return file, nil

}

func parseCPULine(cpuLine *string) (*DBCpuTableEntry, error) {

	dbRow := &DBCpuTableEntry{}
	cpuLineSpilt := strings.Fields(*cpuLine)

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

	layout := "15:04:05"

	parsedTime, err := time.Parse(layout, cpuLineSpilt[0])
	if err != nil {
		log.Fatalf("Error parsing time %v ", err)
		return nil, err
	}
	dbRow.Time = parsedTime

	return dbRow, nil

}

func parseGPULine(gpuLine *string) (*DBGpuTableEntry, error) {
	dbRow := &DBGpuTableEntry{}
	gpuLineSplit := strings.Fields(*gpuLine)

	var err error = nil

	dbRow.UtilizationGpuPercentage, err = strconv.ParseFloat(gpuLineSplit[2], 64)
	dbRow.UtilizationGpuMemory, err = strconv.ParseFloat(gpuLineSplit[4], 64)
	dbRow.MemoryGpuUsedMib, err = strconv.ParseFloat(gpuLineSplit[6], 64)

	layout := "2006/01/02 15:04:05.000"

	gpuLineSplit[1] = gpuLineSplit[1][:len(gpuLineSplit[1])-1]

	combinedString := gpuLineSplit[0] + " " + gpuLineSplit[1]

	parsedTime, err := time.Parse(layout, combinedString)
	if err != nil {
		log.Fatalf("Error parsing time %v ", err)
		return nil, err
	}
	dbRow.Time = parsedTime

	return dbRow, nil
}

func readCPUFile(metric logType, tx *sql.Tx, stmt *sql.Stmt, m *DBManager) error {

	cpuFile, err := openFile(m.slurmPID, metric)
	if err != nil {
		return err
	}
	defer cpuFile.Close()

	cpuScanner := bufio.NewScanner(cpuFile)
	// SKIP THE HEADERS ---
	// Skip 2 header lines in the CPU log
	cpuScanner.Scan()
	cpuScanner.Scan()

	for cpuScanner.Scan() {
		cpuLine := cpuScanner.Text()

		dbRow, err := parseCPULine(&cpuLine)
		if err != nil {
			return err
		}
		dbRow.JobId = *m.slurmPID

		_, err = stmt.Exec(dbRow.JobId, dbRow.Time, dbRow.PID, dbRow.UsrPercentage,
			dbRow.SystemPercentage, dbRow.GuestPercentage, dbRow.WaitPercentage, dbRow.CpuPercentage,
			dbRow.Cpu, dbRow.MinfltsPerS, dbRow.MajfltsPerS, dbRow.VSZ, dbRow.RSS, dbRow.RamPercentage)

		if err != nil {
			tx.Rollback()
			log.Fatalf("Error when buffering metrics: %v", err)
			return err
		}
	}
	return nil
}

func readGPUFile(metric logType, tx *sql.Tx, stmt *sql.Stmt, m *DBManager) error {

	gpuFile, err := openFile(m.slurmPID, metric)
	if err != nil {
		return err
	}
	defer gpuFile.Close()

	gpuScanner := bufio.NewScanner(gpuFile)
	// SKIP THE HEADERS ---
	// Skip 2 header lines in the CPU log
	gpuScanner.Scan()

	for gpuScanner.Scan() {
		cpuLine := gpuScanner.Text()

		dbRow, err := parseGPULine(&cpuLine)
		if err != nil {
			return err
		}
		dbRow.JobId = *m.slurmPID

		_, err = stmt.Exec(dbRow.JobId, dbRow.Time, dbRow.UtilizationGpuPercentage, dbRow.UtilizationGpuMemory, dbRow.MemoryGpuUsedMib)

		if err != nil {
			tx.Rollback()
			log.Fatalf("Error when buffering metrics: %v", err)
			return err
		}
	}
	return nil

}

func readFilesSaveToDb(metric logType, tx *sql.Tx, stmt *sql.Stmt, m *DBManager) error {

	if metric == CPUMetric {
		return readCPUFile(metric, tx, stmt, m)
	} else if metric == GPUMetric {
		return readGPUFile(metric, tx, stmt, m)
	}

	return errors.New("Wrong metric type")
}
