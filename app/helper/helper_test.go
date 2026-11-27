package helper

import (
	"errors"
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

// writeFakeScontrol creates an executable file named scontrol in dir and returns its path.
func writeFakeScontrol(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "scontrol")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindScontrolOrder(t *testing.T) {
	envBin := writeFakeScontrol(t, t.TempDir())
	pathDir := t.TempDir()
	pathBin := writeFakeScontrol(t, pathDir)
	fallbackBin := writeFakeScontrol(t, t.TempDir())
	missing := filepath.Join(t.TempDir(), "missing")
	emptyDir := t.TempDir()

	tests := []struct {
		name      string
		envValue  string // SCONTROL_PATH
		pathValue string // PATH
		fallbacks []string
		want      string
	}{
		{name: "SCONTROL_PATH wins over PATH", envValue: envBin, pathValue: pathDir, fallbacks: []string{fallbackBin}, want: envBin},
		{name: "PATH wins over known locations", pathValue: pathDir, fallbacks: []string{fallbackBin}, want: pathBin},
		{name: "invalid SCONTROL_PATH falls through to PATH", envValue: missing, pathValue: pathDir, fallbacks: []string{fallbackBin}, want: pathBin},
		{name: "first valid known location", pathValue: emptyDir, fallbacks: []string{missing, fallbackBin}, want: fallbackBin},
	}

	for _, test := range tests {
		t.Run(test.name, func(st *testing.T) {
			st.Setenv("SCONTROL_PATH", test.envValue)
			st.Setenv("PATH", test.pathValue)

			got, err := findScontrol(test.fallbacks)
			if err != nil {
				st.Fatalf("findScontrol() error = %v", err)
			}
			if got != test.want {
				st.Errorf("findScontrol() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestFindScontrolNotFound(t *testing.T) {
	dir := t.TempDir()
	notExecutable := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(notExecutable, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")

	t.Setenv("SCONTROL_PATH", "")
	t.Setenv("PATH", t.TempDir())

	_, err := findScontrol([]string{dir, notExecutable, missing})
	if err == nil {
		t.Fatal("findScontrol() error = nil, want an error when nothing is valid")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error should wrap os.ErrNotExist for the missing path, got: %v", err)
	}
	for _, want := range []string{"is a directory", "is not executable"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing reason %q, got: %v", want, err)
		}
	}
}
