package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	slurmPID := flag.String("job-id", "", "Slurm PID")
	accountName := flag.String("account", "", "account name")
	flag.Parse()

	if *slurmPID == "" {
		log.Printf("Slurm PID not found")
		return
	}
	if *accountName == "" {
		log.Printf("Account name not found")
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

	manager.accountName = accountName
	manager.slurmPID = slurmPID

	err = manager.InitDatabase()
	if err != nil {
		return
	}
	fmt.Println("Connection established")

	err = manager.SaveMetricToDB()
	if err != nil {
		return
	}

	fmt.Println("successfully saved logs")
	var cpuFileName string = "usage_cpu_ram_" + *slurmPID + ".log"
	var gpuFileName string = "usage_gpu_" + *slurmPID + ".log"
	fmt.Println("Removing log files")

	err = os.Remove(cpuFileName)
	if err != nil {
		log.Fatalln("Couldn't remove cpu log file")
	}
	err = os.Remove(gpuFileName)
	if err != nil {
		log.Fatalln("Couldn't remove gpu log file")
	}

}
