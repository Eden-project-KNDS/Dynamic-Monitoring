# Active Monitor Application

This directory contains the active Go module for the Slurm node monitor and its Prometheus exporter. It is the runtime that exposes GPU and cgroup-based job metrics from a compute node to Prometheus.

For the current implementation details and known limitations, see [Documentation/DOCUMENTATION.md](../Documentation/DOCUMENTATION.md).

## Requirements

- Go 1.27.2 defined in [go.mod](go.mod)
- NVIDIA drivers and working NVML access
- Linux cgroup v2 and a Slurm job layout compatible with `/sys/fs/cgroup/system.slice/slurmstepd.scope/`
- `scontrol` on `PATH`
- `make`, `goimports`, and `golangci-lint`

## Build and run

```bash
cd app
make build
./slurm-monitor <node-name> <port>
```

The service exposes metrics at `http://<node>:<port>/metrics`. The default local run target is:

```bash
cd app
make run
```

This uses `gpunode3` and port `9840`. The monitor serves one Prometheus scrape at a time and returns `503` for overlapping requests. SIGINT or SIGTERM shuts the HTTP server down cleanly and releases NVML resources.

For the overall deployment and Grafana setup, see [../deploy/README.md](../deploy/README.md).

## Make targets

- `make` runs `fmt` and then `lint`
- `make fmt` formats Go source and imports
- `make lint` runs `golangci-lint`
- `make test` runs the Go test suite
- `make build` builds the `slurm-monitor` binary
- `make run` builds and starts the monitor on `gpunode3:9840`
- `make test-job` submits a short CPU workload through Slurm
- `make test-metrics` checks for `eden_` metrics on the local endpoint
- `make clean` removes the binary and local `.out` files

The Slurm-specific targets require a configured cluster environment with the appropriate permissions.

## Package notes

- [exporter/README.md](exporter/README.md) — Prometheus collector and exported metric contract
- [monitor/README.md](monitor/README.md) — NVML and cgroup data sources
- [stress_test/README.md](stress_test/README.md) — test workloads submitted to Slurm

## Current limitations

This monitor is intentionally node-local and prototype-oriented.