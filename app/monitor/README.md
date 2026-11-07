# Monitor Data Sources

This package supplies GPU and Slurm job data to the Prometheus exporter.

## GPU data

`gpu_monitor.go` initializes NVIDIA's NVML library and reads utilization,
memory, temperature, power, and device UUID from each visible GPU. The exporter
currently exposes utilization and used memory only. If NVML cannot initialize
or no devices are visible, GPU collection returns no GPU samples and the HTTP
server can continue serving job metrics.

## Slurm jobs and cgroups

`cgroups_monitor.go` discovers directories named `job_<id>` under the hard-coded
path `/sys/fs/cgroup/system.slice/slurmstepd.scope/`. It calls `scontrol show job`
to cache job metadata, refreshes discovery every five seconds, and reads CPU and
memory cgroup values when Prometheus scrapes `/metrics`.

The monitor currently depends on Linux cgroup v2 files and the cgroup layout
matching that path. Missing or unreadable cgroup values are returned as zero.
Job discovery is node-local; it does not query the Slurm controller for a full
cluster-wide job list.

## Current limitations

- GPU metrics are not associated with Slurm jobs; their `job_id` label is empty.
- Only a subset of the fields and metrics in the architecture specification is
  currently collected or exported.
- The cgroup path and HTTP listen port are not configurable through flags or
  environment variables.
- The application does not yet coordinate graceful shutdown and NVML cleanup.

For the metric contract and implementation status, see
[`Documentation/DOCUMENTATION.md`](../../Documentation/DOCUMENTATION.md).