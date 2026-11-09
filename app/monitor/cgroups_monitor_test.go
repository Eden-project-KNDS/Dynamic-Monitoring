package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyExists(t *testing.T) {
	jobs := map[string]jobData{"123": {user: "alice"}}

	if !keyExists(jobs, "123") {
		t.Fatal("keyExists returned false for a present key")
	}
	if keyExists(jobs, "456") {
		t.Fatal("keyExists returned true for a missing key")
	}
}

func TestGetJobId(t *testing.T) {
	if got := getJobId("job_12345"); got != "12345" {
		t.Fatalf("getJobId() = %q, want %q", got, "12345")
	}
}

func TestParseScontrolValue(t *testing.T) {
	input := "JobId=123 UserId=alice(1000) Account=research JobName=simulation Partition=compute"
	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "user id with parenthesized uid", key: "UserId", want: "alice(1000)"},
		{name: "job name", key: "JobName", want: "simulation"},
		{name: "missing key", key: "NumCPUs", want: "unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseScontrolValue(input, test.key); got != test.want {
				t.Errorf("parseScontrolValue(%q) = %q, want %q", test.key, got, test.want)
			}
		})
	}
}

func TestReadUint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.current")
	if err := os.WriteFile(path, []byte(" 4096\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readUint(path); got != 4096 {
		t.Errorf("readUint() = %d, want 4096", got)
	}

	if err := os.WriteFile(path, []byte("not-a-number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readUint(path); got != 0 {
		t.Errorf("readUint() for invalid content = %d, want 0", got)
	}

	if got := readUint(filepath.Join(t.TempDir(), "missing")); got != 0 {
		t.Errorf("readUint() for a missing file = %d, want 0", got)
	}
}

func TestReadStatValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.stat")
	content := "usage_usec 1200\nuser_usec 700\nsystem_usec 500\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := readStatValue(path, "usage_usec"); got != 1200 {
		t.Errorf("readStatValue() = %d, want 1200", got)
	}
	if got := readStatValue(path, "missing_key"); got != 0 {
		t.Errorf("readStatValue() for an absent key = %d, want 0", got)
	}
	if got := readStatValue(filepath.Join(t.TempDir(), "missing"), "usage_usec"); got != 0 {
		t.Errorf("readStatValue() for a missing file = %d, want 0", got)
	}
}

func writeCgroupProcs(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
