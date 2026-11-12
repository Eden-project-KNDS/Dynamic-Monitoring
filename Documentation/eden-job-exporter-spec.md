# Eden Job Exporter: Architecture Specification (Draft)

## 1. Goal

Run a daemon continuously on each compute node. Without manual steps, expose
for **every Slurm job**:

- the CPU, RAM, and GPU resources the job **uses**;
- the resources the job **has allocated**.

Prometheus and Grafana can then show where allocated resources are underused.

## 2. Architecture

```mermaid
flowchart LR
    subgraph NODE["Compute node"]
        direction TB
        SD["systemd"] -->|"start, restart"| EXP["eden-job-exporter<br/>/metrics"]
        CG[("Job cgroup<br/>(kernel counters)")]
        NV[("NVML<br/>(NVIDIA driver)")]
        J1["job 1001"] -. "processes in" .-> CG
        J2["job 1002"] -. "processes in" .-> CG
        EXP -->|"CPU / RAM usage"| CG
        EXP -->|"GPU usage"| NV
    end

    CTL["slurmctld"]
    EXP -->|"job metadata and allocation<br/>(once per job)"| CTL

    PROM["Prometheus"] -->|"scrapes /metrics<br/>every 15–30 s"| EXP
    GRAF["Grafana / reports"] --> PROM
```

- **Pull model:** the daemon does not push data; Prometheus scrapes it.
- **Usage** comes from the node (cgroups and NVML); **job metadata and
  allocations** come from Slurm.

## 3. Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Start
    Start --> GPU: NVIDIA driver available
    Start --> CPUonly: no GPU or driver
    GPU --> Running
    CPUonly --> Running
    Running --> Running: serves Prometheus
    Running --> Shutdown: SIGTERM / SIGINT
    Shutdown --> [*]: finish requests,<br/>release NVML
```

- Run under **systemd**: start with the node after Slurm, restart after a crash,
  and run without root privileges.
- **One instance per node:** systemd provides this guarantee.
- On a node **without GPUs**, continue collecting CPU and RAM metrics.

## 4. Job discovery

```mermaid
flowchart TD
    T["periodically in background"] --> L["list jobs on this node"]
    L --> C{"job already known?"}
    C -->|yes| K["no action"]
    C -->|no| S["fetch Slurm metadata:<br/>user, partition, CPU/RAM/GPU allocation"]
    S --> P["cache job metadata"]
    L --> E{"job has ended?"}
    E -->|yes| D["remove from cache"]
```

- Discovery runs in the background, independently of Prometheus, because
  querying Slurm can be slow.
- Query Slurm **once per job**, not for every metric scrape.

## 5. Associate GPUs with jobs

```mermaid
flowchart LR
    subgraph ALLOC["Slurm allocation"]
        A1["job 1001 → GPU 0, 1"]
        A2["job 1002 → GPU 2"]
    end
    subgraph SAMPLE["GPU sampling (NVML)"]
        G0["GPU 0: 95%"]
        G1["GPU 1: 0%  (allocated, idle)"]
        G2["GPU 2: 40%"]
        G3["GPU 3: 0%  (free)"]
    end
    A1 --> G0
    A1 --> G1
    A2 --> G2
```

Associate a GPU with a job **using Slurm's allocation**, not by inspecting
processes on the GPU. An allocated but idle GPU has no processes, so allocation
data is needed to distinguish an *underused* GPU from a *free* one.

## 6. Prometheus scrape

```mermaid
sequenceDiagram
    participant P as Prometheus
    participant E as eden-job-exporter
    participant L as Job cache
    participant N as cgroup + NVML

    P->>E: GET /metrics
    E->>L: Which jobs are on this node?
    loop each job and GPU
        E->>N: Read current usage
    end
    E-->>P: Metrics (usage + allocation)
```

- Read values **when scraped**; the daemon does not need its own sampling loop.
- Export CPU time as a monotonically increasing counter; Prometheus computes
  rates and utilization.

## 7. Metrics

This metric set is a contract for dashboards and reports. Rename metrics only
deliberately.

| Metric | Meaning |
|---|---|
| `eden_job_info` | Job metadata: user, account, partition, and name |
| `eden_job_cpu_usage_seconds_total` | Cumulative CPU time |
| `eden_job_cpus_allocated` | Allocated CPU cores |
| `eden_job_memory_used_bytes` | Current RAM use |
| `eden_job_memory_peak_bytes` | Peak RAM use |
| `eden_job_memory_allocated_bytes` | Allocated RAM |
| `eden_job_gpus_allocated` | Allocated GPUs |
| `eden_job_time_limit_seconds` | Job time limit |
| `eden_job_start_time_seconds` | Job start time |
| `eden_gpu_utilization_ratio` | GPU utilization (0–1), associated with a job |
| `eden_gpu_memory_used_bytes` | GPU memory in use |

Example: a GPU allocated to a job but idle for 30 minutes:

```promql
avg_over_time(eden_gpu_utilization_ratio{job_id!=""}[30m]) < 0.05
```

This is a draft target design. See [the current implementation notes](DOCUMENTATION.md)
for the metrics and behavior that exist in the active application today.
