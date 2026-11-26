package helper

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// GeneralLogger writes to stderr until InitLogger redirects it to a file
var GeneralLogger = log.New(os.Stderr, "[+]General Logger:\t", log.Ldate|log.Ltime|log.Lshortfile)

// ErrorLogger writes to stderr until InitLogger redirects it to a file.
var ErrorLogger = log.New(os.Stderr, "[X]Error Logger:\t", log.Ldate|log.Ltime|log.Lshortfile)

// InitLogger redirects both loggers to general-log.log in $LOGS_DIRECTORY
// If it fails, it returns the error and the loggers keep writing to stderr.
func InitLogger() error {
	logDir := os.Getenv("LOGS_DIRECTORY")
	if logDir == "" {
		logDir = "./log"
	}

	// Ensure the directory exists before opening the file
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("creating log directory %s: %w", logDir, err)
	}

	logPath := filepath.Join(logDir, "general-log.log")
	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file %s: %w", logPath, err)
	}
	GeneralLogger.SetOutput(file)
	ErrorLogger.SetOutput(file)
	return nil
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
