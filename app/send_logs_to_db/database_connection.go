package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DBManager struct {
	DB *sql.DB
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
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS resource_metrics(
			time TIMESTAMPTZ NOT NULL,
			job_id TEXT NOT NULL,
			pid TEXT NOT NULL,
			usr_percentage DOUBLE PRECISION,
			system_percentage DOUBLE PRECISION,
			guest_percentage DOUBLE PRECISION,
			wait_percentage DOUBLE PRECISION,
			cpu_percentage DOUBLE PRECISION,
			cpu INTEGER,
			minflts_per_s DOUBLE PRECISION,
			majflts_per_s DOUBLE PRECISION.
			vsz INTEGER,
			RSS INTEGER,
			ram_percentage DOUBLE PRECISION,
			utilization_gpu_percentage DOUBLE PRECISION,
			utilization_gpy_memory DOUBLE PRECISION,
			memory_gpu_used_mib  DOUBLE PRECISION
			);
	`
	_, err := m.DB.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	createHypertableQuery := `
		SELECT create_hypertable('resource_metrics', 'time', if_not_exists => TRUE);
	`
	_, err = m.DB.Exec(createHypertableQuery)
	if err != nil {
		return fmt.Errorf("failed to create hypertable: %v", err)

	}
	return nil
}
