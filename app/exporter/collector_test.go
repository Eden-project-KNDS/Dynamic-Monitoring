package exporter

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	monitor "slurm-monitor/monitor"
)

// testSource returns exactly the data a test gives it.
type testSource struct {
	gpus []monitor.GPUStat
	jobs []monitor.JobStat
}

func (s testSource) GPUs() []monitor.GPUStat { return s.gpus }
func (s testSource) Jobs() []monitor.JobStat { return s.jobs }

func TestJobMetrics(t *testing.T) {
	src := testSource{
		jobs: []monitor.JobStat{
			{
				JobID: "42", User: "alice", Account: "lab-a", Partition: "gpu", Name: "train",
				CPUUsageUsec:  90_000_000, // 90 s
				CPUsAllocated: 4,
				MemoryCurrent: 1024,
				MemoryPeak:    2048,
				MemoryLimit:   0, // no limit -> no eden_job_memory_allocated_bytes
			},
		},
	}

	expected := `
# HELP eden_job_info Job description from Slurm (value is always 1).
# TYPE eden_job_info gauge
eden_job_info{account="lab-a",job_id="42",job_name="train",partition="gpu",user="alice"} 1
# HELP eden_job_cpu_usage_seconds_total CPU time used by the job.
# TYPE eden_job_cpu_usage_seconds_total counter
eden_job_cpu_usage_seconds_total{job_id="42"} 90
# HELP eden_job_cpus_allocated CPU cores allocated to the job.
# TYPE eden_job_cpus_allocated gauge
eden_job_cpus_allocated{job_id="42"} 4
# HELP eden_job_memory_used_bytes Memory currently used by the job.
# TYPE eden_job_memory_used_bytes gauge
eden_job_memory_used_bytes{job_id="42"} 1024
# HELP eden_job_memory_peak_bytes Peak memory used by the job.
# TYPE eden_job_memory_peak_bytes gauge
eden_job_memory_peak_bytes{job_id="42"} 2048
`

	err := testutil.CollectAndCompare(
		NewCollector(src, src),
		strings.NewReader(expected),
		"eden_job_info",
		"eden_job_cpu_usage_seconds_total",
		"eden_job_cpus_allocated",
		"eden_job_memory_used_bytes",
		"eden_job_memory_peak_bytes",
		"eden_job_memory_allocated_bytes",
	)
	if err != nil {
		t.Error(err)
	}
}
