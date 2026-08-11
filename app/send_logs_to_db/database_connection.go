package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type DBManager struct {
	DB *sql.DB
}

type DBUserTableEntry struct {
	Time    time.Time
	JobId   string
	Account string
}
type DBCpuTableEntry struct {
	JobId            string
	PID              string
	UsrPercentage    float64
	SystemPercentage float64
	GuestPercentage  float64
	WaitPercentage   float64
	CpuPercentage    float64
	Cpu              float64
	MinfltsPerS      float64
	MajfltsPerS      float64
	VSZ              float64
	RSS              float64
	RamPercentage    float64
}
type DBGpuTableEntry struct {
	JobId                    string
	UtilizationGpuPercentage float64
	UtilizationGpuMemory     float64
	MemoryGpuUsedMib         float64
}

func (m *DBManager) connect(connStr *string, driverType *string) error {
	var err error

	m.DB, err = sql.Open(*driverType, *connStr)
	if err != nil {
		log.Fatalf("Error parsing connection string: %v\n", err)
		return err
	}

	err = m.DB.Ping()
	if err != nil {
		log.Fatalf("Error connecting to the db %v\n", err)
		return err
	}
	return nil
}

func (m *DBManager) InitDatabase() error {
	createUserTAble := `
		CREATE TABLE IF NOT EXISTS UserAccount(
			job_id TEXT PRIMARY KEY,
			time TIMESTAMPTZ NOT NULL,
			account TEXT
			);
	`

	createCPUTable := `
		CREATE TABLE IF NOT EXISTS CPUMetrics(
			job_id TEXT NOT NULL REFERENCES UserAccount(job_id) ON DELETE CASCADE,
			pid TEXT,
			usr_percentage DOUBLE PRECISION,
			system_percentage DOUBLE PRECISION,
			guest_percentage DOUBLE PRECISION,
			wait_percentage DOUBLE PRECISION,
			cpu_percentage DOUBLE PRECISION,
			cpu DOUBLE PRECISION,
			minflts_per_s DOUBLE PRECISION,
			majflts_per_s DOUBLE PRECISION,
			vsz DOUBLE PRECISION,
			rss DOUBLE PRECISION,
			ram_percentage DOUBLE PRECISION
			);
	
	`

	createGPUTable := `
		CREATE TABLE IF NOT EXISTS GPUMetric(
		job_id TEXT NOT NULL REFERENCES UserAccount(job_id) ON DELETE CASCADE,
		utilization_gpu_percentage DOUBLE PRECISION,
			utilization_gpu_memory DOUBLE PRECISION,
			memory_gpu_used_mib  DOUBLE PRECISION
			);
	`

	_, err := m.DB.Exec(createUserTAble)
	if err != nil {
		log.Fatalf("failed to create user table: %v", err)
		return err
	}
	_, err = m.DB.Exec(createCPUTable)
	if err != nil {
		log.Fatalf("failed to create cpu table: %v", err)
		return err
	}
	_, err = m.DB.Exec(createGPUTable)
	if err != nil {
		log.Fatalf("failed to create gpu table: %v", err)
		return err
	}

	createHypertableQuery := `
		SELECT create_hypertable('UserAccount', 'time', if_not_exists => TRUE);
	`
	_, err = m.DB.Exec(createHypertableQuery)
	if err != nil {
		log.Fatalf("failed to create hypertable: %v", err)
		return err

	}
	return nil
}

func (m *DBManager) SaveMetricBatch(accountName *string, slurmPID *string) error {

	tx, err := m.DB.Begin()
	if err != nil {
		log.Fatalf("failed to begin transaction %v", err)
		return err
	}
	stmt, err := tx.Prepare(pq.CopyIn("resource_metrics", "time", "job_id",
		"account", "pid", "usr_percentage", "system_percentage", "guest_percentage", "wait_percentage", "cpu_percentage", "cpu",
		"minflts_per_s", "majflts_per_s", "vsz", "rss", "ram_percentage", "utilization_gpu_percentage", "utilization_gpu_memory", "memory_gpu_used_mib"))

	if err != nil {
		tx.Rollback()
		log.Fatalf("failed to prepare copy: %v", err)
		return err
	}
	err = readFilesSaveToDb(accountName, slurmPID, tx, stmt)
	if err != nil {
		return err
	}

	_, err = stmt.Exec()
	if err != nil {
		tx.Rollback()
		log.Fatalf("failed to flush copy statement: %v", err)
		return err
	}

	err = stmt.Close()
	if err != nil {
		tx.Rollback()
		log.Fatalf("failed to close statement %v", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("failed to commit transaction %v", err)
		return err
	}

	return nil

}
