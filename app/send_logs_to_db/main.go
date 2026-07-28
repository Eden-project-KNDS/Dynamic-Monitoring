package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	slurmPID := flag.String("job-id", "", "Slurm PID")
	accountName := flag.String("account", "", "account name")
	flag.Parse()

	if *slurmPID == "" {
		log.Fatal("Slurm PID not found")
		return
	}
	if *accountName == "" {
		log.Fatal("Account name not found")
		return
	}

	connStr := "host=localhost port=5432 user=postgres password=mysecretpassword dbname=postgres sslmode=disable"
	Type := "postgres"

	manager := &DBManager{}

	err := manager.connect(&connStr, &Type)
	if err != nil {
		return
	}
	defer manager.DB.Close()

	err = manager.InitDatabase()
	if err != nil {
		return
	}
	fmt.Println("Connection established")

	err = manager.SaveMetricBatch(accountName, slurmPID)
	if err != nil {
		return
	}

	fmt.Println("successfully saved logs")

}
