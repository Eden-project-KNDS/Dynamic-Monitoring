# Active Monitor Application

This directory contains the active Go module for the Slurm node monitor and its Prometheus exporter. It is the runtime that exposes GPU and cgroup-based job metrics from a compute node to Prometheus.

For the current implementation details and known limitations, see [Documentation/DOCUMENTATION.md](../Documentation/DOCUMENTATION.md).

## Requirements

- Go 1.27.2 defined in [go.mod](go.mod)
- NVIDIA drivers and working NVML access (for GPU metrics)
- Linux cgroup v2 and a Slurm job layout compatible with `/sys/fs/cgroup/system.slice/slurmstepd.scope/`
- `scontrol` for job metadata, found automatically (see [Configuration](#configuration))
- `make`, `goimports`, and `golangci-lint` for development

## Build and run

```bash
cd app
make build
./slurm-monitor <port>        # e.g. ./slurm-monitor 9840
```

The service exposes metrics at `http://<node>:<port>/metrics` on all interfaces. Prometheus adds the `node` label from its scrape configuration. The monitor serves one Prometheus scrape at a time and returns `503` for overlapping requests. SIGINT or SIGTERM shuts the HTTP server down cleanly and releases NVML resources.

To run it as a systemd service, see [Running as a service](#running-as-a-service). For Prometheus and Grafana, see [../deploy/README.md](../deploy/README.md).

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `LISTEN_ADDRESS` | `9840` in [config/config.env](config/config.env) | Port passed to the binary by the systemd unit |
| `SCONTROL_PATH` | unset | Full path to `scontrol`, if it isn't found automatically |
| `LOGS_DIRECTORY` | set by systemd; `./log` otherwise | Directory for the log file |

`scontrol` is looked up in this order: `SCONTROL_PATH`, then `PATH`, then known install locations (`/usr/local/slurm/latest/bin/scontrol` on gpunode3, `/mnt/evafs/software/slurm/amd_epyc/current/bin/scontrol` on stud1, and common system paths). An invalid `SCONTROL_PATH` is logged and skipped. If `scontrol` isn't found, the monitor keeps serving GPU metrics and job metrics are disabled until restart.

Under systemd, `PATH` is minimal, so set `SCONTROL_PATH` in `/etc/slurm-monitor/config.env` if `scontrol` is installed somewhere unusual.

## Logs

The monitor writes to `general-log.log` in `LOGS_DIRECTORY`:

- under systemd: `/var/log/slurm-monitor/general-log.log` (from `LogsDirectory=` in the unit);
- when started by hand: `./log/general-log.log`, relative to the directory you start it from.

It logs the `scontrol` path chosen at startup, one `Scrape served: N GPUs, M jobs in X` line per scrape, new and finished jobs, and read errors. If the log file can't be opened, it logs to stderr instead. Panics and errors before the logger starts always go to stderr, which systemd sends to journald (`journalctl -u slurm-monitor`). The log file isn't rotated yet.

## Running as a service

```bash
cd app
sudo make init_daemon     # first install
sudo make update_daemon   # later updates of the binary
```

- `init_daemon` builds and installs the binary, the unit from [config/slurm-monitor.service](config/slurm-monitor.service) and [config/config.env](config/config.env), then enables and starts the service. It **overwrites** `/etc/slurm-monitor/config.env`.
- `update_daemon` builds, replaces only the binary, and restarts the service. After changing the unit file, copy it and reload systemd yourself:

  ```bash
  sudo cp config/slurm-monitor.service /etc/systemd/system/
  sudo systemctl daemon-reload && sudo systemctl restart slurm-monitor
  ```

## Make targets

- `make` runs `fmt` and then `lint`
- `make fmt` formats Go source and imports (needs `goimports`)
- `make lint` runs `golangci-lint` with [.golangci.yml](.golangci.yml)
- `make test` runs the Go test suite
- `make benchmark` runs the benchmarks in `monitor/`
- `make build` builds the `slurm-monitor` binary
- `make init_daemon` / `make update_daemon` install or update the systemd service (see above)
- `make daemon_status` shows the service status
- `make clean` removes the binary and local `.out` files

## Package notes

- [exporter/README.md](exporter/README.md) — Prometheus collector and exported metric contract
- [monitor/README.md](monitor/README.md) — NVML and cgroup data sources
- [helper/helper.go](helper/helper.go) — file logger and `scontrol` lookup

## Current limitations

This monitor is intentionally node-local and prototype-oriented.
