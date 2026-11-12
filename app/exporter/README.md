# Prometheus Exporter

This package implements the `prometheus.Collector` that exposes monitor data as
`eden_*` metrics. It reads a `GPUSource` and a `JobSource` on each Prometheus
scrape; the source interfaces are defined in `collector.go`.

## Exported metrics

| Metric | Type | Labels | Source |
|---|---|---|---|
| `eden_gpu_utilization_ratio` | Gauge | `gpu`, `uuid`, `job_id` | GPU utilization, converted from percent to a ratio |
| `eden_gpu_memory_used_bytes` | Gauge | `gpu`, `uuid`, `job_id` | GPU memory in bytes |
| `eden_job_info` | Gauge | `job_id`, `user`, `account`, `partition`, `job_name` | Slurm job metadata; value is always `1` |
| `eden_job_cpu_usage_seconds_total` | Counter | `job_id` | Cgroup CPU time converted from microseconds to seconds |
| `eden_job_cpus_allocated` | Gauge | `job_id` | CPU allocation from Slurm |
| `eden_job_memory_used_bytes` | Gauge | `job_id` | Current cgroup memory |
| `eden_job_memory_peak_bytes` | Gauge | `job_id` | Peak cgroup memory, omitted when unavailable or zero |
| `eden_job_memory_allocated_bytes` | Gauge | `job_id` | Cgroup memory limit, omitted when absent |

GPU `job_id` comes from the processes running on the card (PID → cgroup) and
uses the same format as the job metrics (`"61"`, not `"job_61"`), so GPU and job
metrics can be joined on `job_id`. A reserved GPU without a process has an empty
`job_id`. See
[`Documentation/eden-job-exporter-spec.md`](../../Documentation/eden-job-exporter-spec.md)
for proposed metrics that are not all implemented yet.

## Tests and fake data

Run the collector tests from the Go module root:

```bash
cd ..
go test ./exporter
```

`FakeSource` provides synthetic data for development, but the active application
currently constructs the NVML and cgroup-backed sources instead.