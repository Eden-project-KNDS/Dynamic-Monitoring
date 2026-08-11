package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type DBManager struct {
	DB          *sql.DB
	accountName *string
	slurmPID    *string
	ID          int64
}

type DBUserTableEntry struct {
	JobId   string
	Account string
}
type DBCpuTableEntry struct {
	ID               int64
	Time             time.Time
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
	ID                       int64
	Time                     time.Time
	UtilizationGpuPercentage float64
	UtilizationGpuMemory     float64
	MemoryGpuUsedMib         float64
}

type logType int

const (
	GPUMetric logType = iota
	CPUMetric
)

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
		CREATE TABLE IF NOT EXISTS useraccount(
			id BIGSERIAL PRIMARY KEY,
			job_id TEXT UNIQUE,
			account TEXT
			);
	`

	createCPUTable := `
		CREATE TABLE IF NOT EXISTS cpumetric(
			id BIGSERIAL NOT NULL REFERENCES useraccount(id) ON DELETE CASCADE,
			time TIMESTAMPTZ NOT NULL,
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
		CREATE TABLE IF NOT EXISTS gpumetric(
		id BIGSERIAL NOT NULL REFERENCES useraccount(id) ON DELETE CASCADE,
		time TIMESTAMPTZ NOT NULL,
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

	createHypertableQueryGPU := `
		SELECT create_hypertable('gpumetric', 'time', if_not_exists => TRUE);
	`
	_, err = m.DB.Exec(createHypertableQueryGPU)
	if err != nil {
		log.Fatalf("failed to create on GPU hypertable: %v", err)
		return err
	}

	createHypertableQueryCPU := `
		SELECT create_hypertable('cpumetric', 'time', if_not_exists => TRUE);
	`
	_, err = m.DB.Exec(createHypertableQueryCPU)
	if err != nil {
		log.Fatalf("failed to create on CPU hypertable: %v", err)
		return err
	}

	return nil
}

func (m *DBManager) PrepareTx(metric logType, tx *sql.Tx) (*sql.Stmt, error) {

	if metric == GPUMetric {
		stmt, err := tx.Prepare(pq.CopyIn("gpumetric", "job_id", "time",
			"utilization_gpu_percentage", "utilization_gpu_memory", "memory_gpu_used_mib"))

		return stmt, err
	} else if metric == CPUMetric {
		stmt, err := tx.Prepare(pq.CopyIn("cpumetric", "job_id", "time",
			"pid", "usr_percentage", "system_percentage", "guest_percentage", "wait_percentage", "cpu_percentage", "cpu",
			"minflts_per_s", "majflts_per_s", "vsz", "rss", "ram_percentage"))

		return stmt, err
	}
	return nil, errors.New("Couldn't create stmt")
}

func (m *DBManager) SaveMetricBatch(metric logType) error {

	tx, err := m.DB.Begin()
	if err != nil {
		log.Fatalf("failed to begin transaction %v", err)
		return err
	}

	stmt, err := m.PrepareTx(metric, tx)

	if err != nil {
		tx.Rollback()
		log.Fatalf("failed to prepare copy: %v", err)
		return err
	}
	err = readFilesSaveToDb(metric, tx, stmt, m)
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

func (m *DBManager) SaveRowToUserTable() (int64, error) {

	query := `
	INSERT INTO "useraccount"(job_id, account)
	VALUES ($1, $2)
	RETURNING id;`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var insertedID int64

	err := m.DB.QueryRowContext(ctx, query, m.slurmPID, m.accountName).Scan(&insertedID)

	if err != nil {
		log.Printf("Error saving User data to the database: %v \n", err)
		return 0, err
	}

	return insertedID, nil
}

func (m *DBManager) SaveMetricToDB() error {
	fmt.Println("Started saving metrics to DB")
	const METRIC_SIZE int = 2

	errChan := make(chan error, METRIC_SIZE)

	id, err := m.SaveRowToUserTable()
	if err != nil {
		return err
	}
	m.ID = id
	go func() {
		errChan <- m.SaveMetricBatch(GPUMetric)
	}()

	go func() {
		errChan <- m.SaveMetricBatch(CPUMetric)
	}()

	for i := 0; i < METRIC_SIZE; i++ {
		err := <-errChan
		if err != nil {
			return err
		}

	}
	fmt.Println("All background tasks have finished")

	return nil
}
