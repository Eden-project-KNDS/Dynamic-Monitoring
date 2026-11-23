package monitor

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// GeneralLogger exported
var GeneralLogger *log.Logger

// ErrorLogger exported
var ErrorLogger *log.Logger

func InitLogger() {
	logDir := os.Getenv("LOGS_DIRECTORY")
	if logDir == "" {
		logDir = "./log"
	}

	// Ensure the directory exists before opening the file
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v\n", logDir, err)
		os.Exit(1)
	}

	logPath := filepath.Join(logDir, "general-log.log")
	generalLog, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file %s: %v\n", logPath, err)
		os.Exit(1)
	}
	GeneralLogger = log.New(generalLog, "[+]General Logger:\t", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(generalLog, "[X]Error Logger:\t", log.Ldate|log.Ltime|log.Lshortfile)
}

func isExec(mode fs.FileMode) bool {
	return mode&0111 != 0
}

func FindScontrol() (string, error) {

	path, err := exec.LookPath("scontrol")
	if err != nil {
		ErrorLogger.Printf("Couldn't find scontrol path checking default path")
	} else {
		return path, nil
	}
	fallbackPaths := []string{
		"/usr/local/slurm/latest/bin/scontrol",
		"/opt/slurm/bin/scontrol",
		"/usr/local/bin/scontrol",
		"/usr/bin/scontrol",
		"/mnt/evafs/software/slurm/amd_epyc/current/bin/scontrol",
	}

	for _, fPath := range fallbackPaths {
		file, err := os.Stat(fPath)
		if err != nil {
			ErrorLogger.Printf("%s is not valid scontrol path, error : %v\n", fPath, err)
		} else if file.IsDir() || !isExec(file.Mode().Perm()) {
			ErrorLogger.Printf("%s is not a valid executable file\n", fPath)
		} else {
			GeneralLogger.Printf("Found scontrol path: %s\n", fPath)
			return fPath, nil
		}
	}
	return "", fmt.Errorf("Scontrol is not present\n")

}
