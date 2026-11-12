# Dynamic Monitoring

This repository contains the active Slurm node monitoring stack for the Eden cluster. It gathers GPU activity, cgroup-based CPU and memory usage, and Slurm job metadata, then exposes the data as Prometheus metrics for dashboarding and analysis.

## Overview

The active application lives under [app](app). It is a Go-based exporter that runs on a compute node and serves a Prometheus-compatible metrics endpoint at `/metrics`. The stack is designed for Linux Slurm nodes with NVIDIA GPUs and cgroup v2 enabled.

The repository also includes:

- [deploy](deploy) for the Prometheus and Grafana stack
- [Documentation](Documentation) for implementation notes and architecture drafts
- [perf](perf) for profiling and performance analysis notes
- [_app [deprecated]](_app%20%5Bdeprecated%5D) for an older implementation that is no longer used

## Repository layout

```text
.
├── app/                         Active Go monitor and Prometheus exporter
│   ├── exporter/                Prometheus collector and tests
│   ├── monitor/                 NVML and Slurm/cgroup data sources
│   ├── config/                  Daemon and service configuration
│   ├── stress_test/             Slurm workload helpers
│   ├── Makefile                Build, test, and cluster checks
│   └── go.mod                  Go module definition
├── deploy/                      Prometheus + Grafana via Docker Compose
├── Documentation/               Implementation notes and architecture drafts
├── perf/                        Profiling and performance analysis notes
├── .github/                     CI and repository automation
├── LICENSE                     Project license
├── requirements.txt            Python dependency file for tooling/test helpers
├── commitlint.config.js        Commit convention configuration
├── README.md                   This project overview
└── _app [deprecated]/          Legacy implementation kept for reference
```

## Requirements

- Go 1.27.2 (see [app/go.mod](app/go.mod))
- NVIDIA drivers and working NVML access for GPU metrics
- Linux with cgroup v2 and a Slurm node layout matching the app expectations
- `scontrol` available on `PATH` for job metadata lookup
- `make`, `goimports`, and `golangci-lint` for the app Makefile targets

The cgroup discovery path is currently hard-coded in [app/monitor/cgroups_monitor.go](app/monitor/cgroups_monitor.go), so this is a node-local monitor rather than a general-purpose desktop monitor.

## Quick start

```bash
cd app
make build
./slurm-monitor <node-name> <port>
```

The app serves metrics at `http://<node>:<port>/metrics`. The default local test command in the Makefile is:

```bash
cd app
make run
```

This starts the exporter on `gpunode3:9840` by default. To view the metrics in Grafana or Prometheus, follow the instructions in [deploy/README.md](deploy/README.md).

## What the exporter exposes

The active collector exports `eden_*` metrics such as:

- GPU utilization and memory usage
- Slurm job metadata (`eden_job_info`)
- CPU time and CPU allocation for each job
- cgroup memory usage and peak values

These metrics are produced by the collector in [app/exporter/collector.go](app/exporter/collector.go) and backed by the sources in [app/monitor](app/monitor).

## Deployment

The deployment stack lives under [deploy](deploy). It runs:

- Prometheus for time-series storage
- Grafana for dashboards and visualization

The stack is configured through Docker Compose and expects the monitor to already be running on the target Slurm node.

## Documentation index

- [app/README.md](app/README.md) — active application usage and Make targets
- [Documentation/README.md](Documentation/README.md) — guide to project notes and design docs
- [Documentation/DOCUMENTATION.md](Documentation/DOCUMENTATION.md) — implementation guide for the current code
- [Documentation/eden-job-exporter-spec.md](Documentation/eden-job-exporter-spec.md) — draft architecture and target metric contract
- [Documentation/Prace.md](Documentation/Prace.md) — earlier project notes retained for reference

## Current limitations

- GPU-to-job attribution is based on processes currently running on the GPU; reserved but idle GPUs may still appear with an empty `job_id`.
- The cgroup path is hard-coded and assumes a specific Slurm/cgroup layout.
- The project is built for a compute node environment and is not a general-purpose local monitoring solution.

## Contributing

Please keep changes focused, document behavior in the relevant README or implementation guide, and validate the affected app or deployment commands before merging.
