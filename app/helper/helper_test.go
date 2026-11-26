package helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// restoreLoggers puts both loggers back on stderr when the test ends.
func restoreLoggers(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		GeneralLogger.SetOutput(os.Stderr)
		ErrorLogger.SetOutput(os.Stderr)
	})
}

func TestInitLoggerWritesToFile(t *testing.T) {
	restoreLoggers(t)
	dir := filepath.Join(t.TempDir(), "logs") // doesn't exist yet: InitLogger must create it
	t.Setenv("LOGS_DIRECTORY", dir)

	if err := InitLogger(); err != nil {
		t.Fatalf("InitLogger() error = %v", err)
	}
	GeneralLogger.Print("first general line")
	ErrorLogger.Print("first error line")

	// A restart must append to the file, not overwrite it.
	if err := InitLogger(); err != nil {
		t.Fatalf("second InitLogger() error = %v", err)
	}
	GeneralLogger.Print("after restart")

	data, err := os.ReadFile(filepath.Join(dir, "general-log.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[+]General Logger:", "first general line",
		"[X]Error Logger:", "first error line",
		"after restart",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("log file is missing %q, got:\n%s", want, data)
		}
	}
}

func TestInitLoggerKeepsStderrOnError(t *testing.T) {
	restoreLoggers(t)
	notADir := filepath.Join(t.TempDir(), "plain-file")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOGS_DIRECTORY", notADir)

	if err := InitLogger(); err == nil {
		t.Fatal("InitLogger() error = nil, want an error when LOGS_DIRECTORY is a file")
	}
	if GeneralLogger.Writer() != os.Stderr || ErrorLogger.Writer() != os.Stderr {
		t.Error("loggers should keep writing to stderr after a failed InitLogger()")
	}
}
