package helper

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// defaultScontrolPaths are checked when SCONTROL_PATH is unset and scontrol isn't in PATH.
var defaultScontrolPaths = []string{
	"/usr/local/slurm/latest/bin/scontrol", // gpunode3
	"/opt/slurm/bin/scontrol",
	"/usr/local/bin/scontrol",
	"/usr/bin/scontrol",
	"/mnt/evafs/software/slurm/amd_epyc/current/bin/scontrol", // stud1
}

// checkExecutable return nil if path is a file with an execute permission bit set.
func checkExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%s is not executable", path)
	}
	return nil
}

// FindScontrol returns the path of the scontrol binary. It checks, in order:
// the SCONTROL_PATH variable, the PATH variable, then known install locations.
func FindScontrol() (string, error) {
	return findScontrol(defaultScontrolPaths)
}

func findScontrol(fallbackPath []string) (string, error) {
	if envPath := strings.TrimSpace(os.Getenv("SCONTROL_PATH")); envPath != "" {
		if err := checkExecutable(envPath); err != nil {
			ErrorLogger.Printf("Ignoring SCONTROL_PATH: %v", err)
		} else {
			return envPath, nil
		}
	}

	if path, err := exec.LookPath("scontrol"); err == nil {
		return path, nil
	}

	var errs []error
	for _, path := range fallbackPath {
		err := checkExecutable(path)
		if err == nil {
			return path, nil
		}
		errs = append(errs, err)
	}
	return "", fmt.Errorf("scontrol not found in SCONTROL_PATH, PATH or known locations: %w", errors.Join(errs...))
}
