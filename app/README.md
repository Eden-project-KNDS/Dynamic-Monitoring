# Active Monitor Application

This directory contains the active Go module for the Slurm node monitor and its
Prometheus exporter. For a detailed description of current behavior and known
limitations, see [the implementation documentation](../Documentation/DOCUMENTATION.md).

## Build and run

```bash
make build
./slurm-monitor <node-name> <port>
```

The process exposes metrics at `http://localhost:<port>/metrics` (`make run`
uses `gpunode3` and port `9840`). It serves one scrape at a time; a parallel
request gets `503`. SIGINT or SIGTERM stops the HTTP server and releases NVML.
For Prometheus and Grafana, see [the deployment guide](../deploy/README.md). It requires
access to NVIDIA NVML for GPU metrics and, on Slurm nodes, the configured cgroup
v2 path plus `scontrol` for job discovery. The cgroup path is currently fixed in
`monitor/cgroups_monitor.go`.

## Make targets

- `make` runs `fmt` and then `lint`.
- `make fmt` formats Go source and imports.
- `make lint` runs `golangci-lint`.
- `make test` runs the Go test suite.
- `make build` builds `slurm-monitor`.
- `make test-job` submits a short CPU workload through Slurm.
- `make test-metrics` checks for Eden metrics at the local endpoint.
- `make clean` removes the binary and local `.out` files.

The Slurm targets require a configured cluster environment. The app is a
prototype: a GPU is associated with a job only while the job has a process on
the card, and the implementation does not yet include every metric in the
architecture spec.

See [exporter](exporter/README.md), [monitor](monitor/README.md), and
[stress workload](stress_test/README.md) for package-specific notes.