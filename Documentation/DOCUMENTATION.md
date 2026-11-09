# Current Implementation

## Scope

This document describes the active Go module in `app/`. It is a Slurm
node monitor that exposes selected NVIDIA GPU and job cgroup data through a
Prometheus endpoint.

## Application structure

- `main.go` parses the node-name and port arguments, creates the data sources,
	registers the collector, serves `/metrics` (one scrape at a time), and shuts
	down cleanly on SIGINT or SIGTERM.
- `synchronization/sync.go` holds the shared context and wait group used for
	shutdown.
- `monitor/gpu_monitor.go` initializes NVML and reads GPU statistics when
	Prometheus scrapes the endpoint.
- `monitor/cgroups_monitor.go` discovers local Slurm jobs and reads cgroup v2
	counters and memory values.
- `exporter/collector.go` converts monitor snapshots into Prometheus metrics.
- `../deploy/` runs Prometheus and Grafana; see
	[the deployment guide](../deploy/README.md).
- `Makefile` defines formatting, lint, test, build, and cluster test targets.
- `go.mod` defines the Go module `slurm-monitor` and its dependencies.

## Build and run

The executable requires two arguments, the node name and the listen port:

```bash
cd app
make build
./slurm-cgroup-monitor <node-name> <port>
```

The process listens on `:<port>` on all interfaces; `make run` uses port
`9840`. The node name is not exported by the monitor; Prometheus adds a `node`
label from its scrape configuration (`deploy/prometheus/prometheus.yml`).

### Make targets

- `make` runs `fmt` followed by `lint`.
- `make fmt` runs `go fmt ./...` and `goimports -w .`.
- `make lint` runs `golangci-lint run`.
- `make test` runs `go test -v ./...`.
- `make build` builds `slurm-cgroup-monitor` with the configured CGO warning flag.
- `make test-job` submits a short CPU workload using Slurm.
- `make test-metrics` scrapes the local endpoint and filters Eden metrics.
- `make clean` removes the executable and local `.out` files.

The Slurm-specific targets require a configured cluster and suitable job
submission permissions.

## Collection behavior

Job discovery runs during each Prometheus scrape; there is no background loop.
The monitor scans `/sys/fs/cgroup/system.slice/slurmstepd.scope/` for
directories named `job_<id>`, then runs `scontrol show job <id>` for new jobs to
cache the job's user, account, partition, name, and allocated CPU count. When a
job disappears from that directory, its cached metadata is removed. Because a
scrape updates these maps, the HTTP handler serves one scrape at a time
(`MaxRequestsInFlight: 1`); a parallel request receives `503` instead of
racing on the maps.

The exporter reads each cached job's cgroup values during a scrape. It exports
CPU usage from `cpu.stat` and memory values from `memory.current`,
`memory.peak`, and `memory.max`. Missing files and unparsable values are treated
as zero; peak and allocated-memory metrics are omitted when their values are
zero.

GPU collection uses NVIDIA's `go-nvml` bindings. Utilization is exported as a
ratio and memory use as bytes. Temperature and power are read by the monitor
but are not currently exported. A GPU's `job_id` is taken from the processes
running on it (`/proc/<pid>/cgroup`, with the `job_` prefix stripped), so it
matches `job_id` in the job metrics. If NVML cannot initialize or no GPU is
available, GPU metrics are absent while the HTTP endpoint can still serve job
metrics.

## Exported metric set

| Metric | Meaning |
|---|---|
| `eden_gpu_utilization_ratio` | GPU utilization from 0 to 1 |
| `eden_gpu_memory_used_bytes` | GPU memory in use |
| `eden_job_info` | Job metadata labels; value is always 1 |
| `eden_job_cpu_usage_seconds_total` | Cumulative cgroup CPU time |
| `eden_job_cpus_allocated` | CPU cores allocated to the job |
| `eden_job_memory_used_bytes` | Current cgroup memory |
| `eden_job_memory_peak_bytes` | Peak cgroup memory, when available |
| `eden_job_memory_allocated_bytes` | Cgroup memory limit, when configured |

A reserved GPU with no running process has an empty `job_id`; mapping GPUs to
jobs from the Slurm allocation, GPU count allocation, job time limits, and
several other metrics proposed by the architecture specification are not
implemented.

## Requirements and limitations

- The Go version is declared as `1.27.2` in `app/go.mod`.
- GPU metrics require an NVIDIA driver and working NVML access.
- Job metrics require Linux cgroup v2, the expected cgroup layout, and
	`scontrol` on `PATH`.
- The cgroup path is hard-coded; the listen port is a CLI argument.
- The node-name argument is not included in exported metrics.
- The monitor itself does not persist data. Prometheus in `deploy/` stores the
	scraped metrics (30-day retention) and Grafana displays them.

See the [exporter guide](../app/exporter/README.md),
[monitor guide](../app/monitor/README.md), and
[draft architecture specification](eden-job-exporter-spec.md) for related
details. The specification is a target design, not a feature checklist for the
current binary.