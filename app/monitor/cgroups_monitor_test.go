package monitor

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	helper "slurm-monitor/helper"
	"strconv"
	"strings"
	"sync"
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
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{name: "user id with parenthesized uid", key: "UserId", want: "alice(1000)"},
		{name: "job name", key: "JobName", want: "simulation"},
		{name: "missing key", key: "NumCPUs", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := tryParseScontrolValue(input, test.key)
			if (err != nil) != test.wantErr {
				t.Fatalf("tryParseScontrolValue(%q) error = %v, wantErr %v", test.key, err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("tryParseScontrolValue(%q) = %q, want %q", test.key, got, test.want)
			}
		})
	}
}

func TestTryReadCGroupsAsUint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.current")
	if err := os.WriteFile(path, []byte(" 4096\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := tryReadCGroupsAsUint(path); err != nil || got != 4096 {
		if err == nil {
			t.Errorf("tryReadCGroupsAsUint() = %d, want 4096", got)
		} else {
			t.Errorf("tryReadCGroupsAsUint() parse error")
		}
	}
	if err := os.WriteFile(path, []byte("not-a-number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tryReadCGroupsAsUint(path); !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("tryReadCGroupsAsUint() error = %v, want wrapped syntax error", err)
	}
	if _, err := tryReadCGroupsAsUint(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("tryReadCGroupsAsUint() error = %v, want wrapped missing-file error", err)
	}
}

func TestTryReadStatValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.stat")
	content := "usage_usec 1200\nuser_usec 700\nsystem_usec 500\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if got, err := tryReadStatValue(path, "usage_usec"); err != nil || got != 1200 {
		if err == nil {
			t.Errorf("tryReadStatValue() = %d, want 1200", got)
		} else {
			t.Errorf("tryReadStatValue() parse error")
		}
	}
	if _, err := tryReadStatValue(path, "missing_key"); err == nil {
		t.Errorf("tryReadStatValue() didn't bail for an absent key")
	}
	if _, err := tryReadStatValue(filepath.Join(t.TempDir(), "missing"), "usage_usec"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("tryReadStatValue() error = %v, want wrapped missing-file error", err)
	}
	if err := os.WriteFile(path, []byte("usage_usec not-a-number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tryReadStatValue(path, "usage_usec"); !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("tryReadStatValue() error = %v, want wrapped syntax error", err)
	}
}

func TestReadJobStatIndependentFields(t *testing.T) {
	dir := t.TempDir()
	for file, content := range map[string]string{
		"memory.current": "0\n",
		"memory.max":     "17\n",
		"cpu.stat":       "usage_usec not-a-number\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := readJobStat("123", jobData{user: "alice", cpus: 4}, dir+string(os.PathSeparator))
	if got.JobID != "123" || got.User != "alice" || got.CPUsAllocated != 4 {
		t.Fatalf("readJobStat() lost job metadata: %+v", got)
	}
	if !got.MemoryCurrentValid || got.MemoryCurrent != 0 || !got.MemoryLimitValid || got.MemoryLimit != 17 {
		t.Errorf("readJobStat() lost valid readings: %+v", got)
	}
	if got.CPUUsageValid || got.MemoryPeakValid {
		t.Errorf("readJobStat() marked unavailable readings as valid: %+v", got)
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

func TestTryReadMemoryLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory.max")

	for _, test := range []struct {
		content string
		want    uint64
	}{
		{content: "max\n", want: 0}, // cgroup v2: no limit
		{content: "1073741824\n", want: 1073741824},
	} {
		if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := tryReadMemoryLimit(path)
		if err != nil || got != test.want {
			t.Errorf("tryReadMemoryLimit(%q) = (%d, %v), want (%d, nil)", test.content, got, err, test.want)
		}
	}

	if err := os.WriteFile(path, []byte("unlimited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tryReadMemoryLimit(path); !errors.Is(err, strconv.ErrSyntax) {
		t.Errorf("tryReadMemoryLimit(invalid) error = %v, want wrapped syntax error", err)
	}
	if _, err := tryReadMemoryLimit(filepath.Join(dir, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("tryReadMemoryLimit(missing) error = %v, want wrapped missing-file error", err)
	}
}

func TestReadJobStatQuietOnNoLimitAndNoPeak(t *testing.T) {
	var logs bytes.Buffer
	helper.GeneralLogger.SetOutput(&logs)
	helper.ErrorLogger.SetOutput(&logs)
	t.Cleanup(func() {
		helper.GeneralLogger.SetOutput(os.Stderr)
		helper.ErrorLogger.SetOutput(os.Stderr)
	})
	peakMissingOnce = sync.Once{} // another test may have used it already

	// A job on gpunode3: no memory limit and no memory.peak (kernel 5.15).
	dir := t.TempDir()
	for file, content := range map[string]string{
		"memory.current": "4096\n",
		"memory.max":     "max\n",
		"cpu.stat":       "usage_usec 1000\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	for range 3 { // three scrapes
		got := readJobStat("144", jobData{}, dir+string(os.PathSeparator))
		if !got.MemoryLimitValid || got.MemoryLimit != 0 {
			t.Errorf("memory.max=max should be a valid 0 (no limit), got %+v", got)
		}
		if got.MemoryPeakValid {
			t.Errorf("missing memory.peak should be invalid, got %+v", got)
		}
	}

	out := logs.String()
	if n := strings.Count(out, "memory.peak is missing"); n != 1 {
		t.Errorf("want the missing memory.peak logged once over 3 scrapes, got %d:\n%s", n, out)
	}
	if strings.Contains(out, "memory.max") || strings.Contains(out, "Error Logger") {
		t.Errorf("no errors expected for a job without a memory limit, got:\n%s", out)
	}
}
