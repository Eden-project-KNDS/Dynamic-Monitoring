# Dynamic Monitoring Monitor

This folder contains a small Go-based monitor utility for collecting CPU/RAM and GPU usage data for a running process.

## What it does

The monitor starts two background collectors:
- `pidstat` for CPU and memory usage
- `nvidia-smi` for GPU usage statistics

It writes the results to:
- `usage_cpu_ram.log`
- `usage_gpu.log`

## Build

From this directory, build the monitor with:

```bash
go build -o eden-monitor monitor.go
```

## How to use monitor.go

Run the monitor with the target process name and a job identifier:

```bash
./eden-monitor --job-id <job-id> --name <process-name>
```

Example:

```bash
./eden-monitor --job-id $SLURM_JOB_ID --name main
```

### Requirements

Make sure the following tools are available on the system:
- `pidstat`
- `nvidia-smi`
- Go

The monitor stops cleanly when it receives `SIGINT` or `SIGTERM`.
