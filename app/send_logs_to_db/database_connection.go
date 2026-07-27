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
			cpu_usage DOUBLE PRECISION,
			ram_usage DOUBLE PRECISION
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
