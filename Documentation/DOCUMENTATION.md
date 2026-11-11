# Current implementation

This document describes the active monitor in [app](../app). The application is a Go-based Slurm node exporter that reads NVIDIA GPU telemetry and cgroup data, then exposes `eden_*` metrics through a Prometheus endpoint.

## Application structure

- [app/main.go](../app/main.go) parses the node name and port, registers the collector, serves `/metrics`, and shuts down cleanly on `SIGINT` or `SIGTERM`.
- [app/synchronization/sync.go](../app/synchronization/sync.go) contains the shared shutdown context and wait group.
- [app/monitor/gpu_monitor.go](../app/monitor/gpu_monitor.go) initializes NVML and reads GPU values when Prometheus scrapes the endpoint.
- [app/monitor/cgroups_monitor.go](../app/monitor/cgroups_monitor.go) discovers local Slurm jobs and reads cgroup v2 counters and memory values.
- [app/exporter/collector.go](../app/exporter/collector.go) converts the monitor snapshots into Prometheus metrics.
- [app/Makefile](../app/Makefile) defines formatting, lint, test, build, and Slurm test targets.
- [app/go.mod](../app/go.mod) defines the Go module name and dependencies (`slurm-monitor`).
- [deploy](../deploy) runs Prometheus and Grafana for persistence and visualization.

## Runtime requirements

- Go version: `1.27.2` (see [app/go.mod](../app/go.mod))
- NVIDIA driver and working NVML access for GPU metrics
- Linux cgroup v2 on the node and a Slurm job layout matching `/sys/fs/cgroup/system.slice/slurmstepd.scope/`
- `scontrol` available on `PATH`
- `make`, `goimports`, and `golangci-lint` for local development

The cgroup path is intentionally hard-coded in [app/monitor/cgroups_monitor.go](../app/monitor/cgroups_monitor.go), which means this is a node-local Slurm monitor rather than a generic host monitor.

## Build and run

The executable requires two arguments: the node name and the listen port.

```bash
cd app
make build
./slurm-monitor <node-name> <port>
```

The service binds to `:<port>` on all interfaces. The local Makefile target uses `gpunode3` and port `9840`:

```bash
cd app
make run
```

The node name is not exported within the metrics; Prometheus adds a `node` label from its scrape configuration in [deploy/prometheus/prometheus.yml](../deploy/prometheus/prometheus.yml).

### Make targets

- `make` runs `fmt` and then `lint`
- `make fmt` runs `gofmt` and `goimports`
- `make lint` runs `golangci-lint`
- `make test` runs `go test -v ./...`
- `make build` builds the `slurm-monitor` binary
- `make run` builds and starts the monitor on `gpunode3:9840`
- `make test-job` submits a short CPU workload through Slurm
- `make test-metrics` scrapes the local endpoint and checks for `eden_` metrics
- `make clean` removes the binary and `.out` files

The Slurm-specific targets require a configured cluster environment and suitable job submission permissions.

## Collection behavior

Job discovery is performed during each Prometheus scrape; there is no background job loop. The monitor scans `/sys/fs/cgroup/system.slice/slurmstepd.scope/` for directories named `job_<id>`, then calls `scontrol show job <id>` for new jobs to cache job metadata such as user, account, partition, name, and allocated CPU count. When a job disappears from the cgroup tree, its cached metadata is removed.

Because the job maps are updated during a scrape, the HTTP handler allows only one in-flight scrape (`MaxRequestsInFlight: 1`); parallel requests receive `503` instead of racing on shared state.

The exporter reads each cached job's cgroup values during a scrape. It exports CPU usage from `cpu.stat` and memory values from `memory.current`, `memory.peak`, and `memory.max`. Missing files and unparsable values are treated as zero, and peak or allocated memory values are omitted when the value resolves to zero.

GPU collection uses NVIDIA's `go-nvml` bindings. Utilization is exported as a ratio and memory usage as bytes. Temperature and power are read by the monitor but are not exported currently. A GPU's `job_id` is derived from the processes running on it by reading `/proc/<pid>/cgroup` and stripping the `job_` prefix. If NVML cannot initialize or no GPU is available, GPU metrics are absent while the HTTP endpoint continues serving job metrics.

## Exported metric set

| Metric | Meaning |
|---|---|
| `eden_gpu_utilization_ratio` | GPU utilization from 0 to 1 |
| `eden_gpu_memory_used_bytes` | GPU memory in use |
| `eden_job_info` | Job metadata labels; value is always `1` |
| `eden_job_cpu_usage_seconds_total` | Cumulative cgroup CPU time |
| `eden_job_cpus_allocated` | CPU cores allocated to the job |
| `eden_job_memory_used_bytes` | Current cgroup memory |
| `eden_job_memory_peak_bytes` | Peak cgroup memory, when available |
| `eden_job_memory_allocated_bytes` | Cgroup memory limit, when configured |

A reserved GPU with no active process has an empty `job_id`. Mapping GPUs to jobs from Slurm allocation, GPU count, time limits, and several other target metrics described in the draft architecture are not yet implemented.

## Requirements and limitations

- The Go version is declared as `1.27.2` in [app/go.mod](../app/go.mod).
- GPU metrics require NVIDIA drivers and NVML access.
- Job metrics require Linux cgroup v2, the expected cgroup layout, and `scontrol` on `PATH`.
- The cgroup path is hard-coded; the listen port is provided as a CLI argument.
- The node-name argument is not included in exported metrics.
- The monitor does not persist data itself; Prometheus in [deploy](../deploy) stores the scraped series and Grafana renders the dashboard.

For package-level details, see [app/exporter/README.md](../app/exporter/README.md), [app/monitor/README.md](../app/monitor/README.md), and the draft architecture in [eden-job-exporter-spec.md](eden-job-exporter-spec.md). The specification is a design target rather than a checklist of features already implemented in the active binary.