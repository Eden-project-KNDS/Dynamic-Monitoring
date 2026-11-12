# Monitor Data Sources

This package supplies GPU and Slurm job data to the Prometheus exporter.

## GPU data

`gpu_monitor.go` initializes NVIDIA's NVML library and reads utilization,
memory, temperature, power, and device UUID from each visible GPU. A GPU's job
is found from the processes running on it: `/proc/<pid>/cgroup` contains the
job's `job_<id>` directory, and the `job_` prefix is stripped. The exporter
currently exposes utilization and used memory only. If NVML cannot initialize
or no devices are visible, GPU collection returns no GPU samples and the HTTP
server can continue serving job metrics.

## Slurm jobs and cgroups

`cgroups_monitor.go` discovers directories named `job_<id>` under the hard-coded
path `/sys/fs/cgroup/system.slice/slurmstepd.scope/`. It calls `scontrol show job`
to cache job metadata, and reads CPU and memory cgroup values. Both discovery
and reading happen during each Prometheus scrape of `/metrics`; there is no
background loop. Because discovery updates the job maps, the HTTP handler
allows only one scrape at a time (`MaxRequestsInFlight: 1` in `main.go`).

The monitor currently depends on Linux cgroup v2 files and the cgroup layout
matching that path. Missing or unreadable cgroup values are returned as zero.
Job discovery is node-local; it does not query the Slurm controller for a full
cluster-wide job list.

## Current limitations

- A reserved GPU without a running process has an empty `job_id`, so an idle
  reserved GPU cannot yet be told apart from a free one.
- Only a subset of the fields and metrics in the architecture specification is
  currently collected or exported.
- The cgroup path is hard-coded; the HTTP port is the second CLI argument.

For the metric contract and implementation status, see
[`Documentation/DOCUMENTATION.md`](../../Documentation/DOCUMENTATION.md).