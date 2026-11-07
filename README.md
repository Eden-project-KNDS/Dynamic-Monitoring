# Eden Cluster Monitor

A monitoring and resource optimization system for the GPU cluster at the Faculty of Mathematics and Information Science (MiNI), Warsaw University of Technology.

---

## Problem

Research teams and students using the Eden cluster (32× A100, 8× H100, 4× P100) routinely overprovision resources — reserving 72-hour time limits for jobs that finish in 4 hours, or allocating 4 GPUs when only 1 is used. This blocks resources for other users without any real need.

The existing monitoring solution generates thousands of log files per month and provides no tooling to help users make better decisions before submitting a job.

---

## What we're building

The current prototype is a Go-based node monitor for Slurm compute nodes. It
exposes GPU utilization and memory, plus discovered job metadata and CPU/RAM
usage, through a Prometheus `/metrics` endpoint.

The longer-term goal is to help users right-size resource requests and help
cluster administrators identify underused allocations. Planned work includes
GPU-to-job attribution, broader resource metrics, historical job analysis, and
recommendations for Slurm job settings. These are roadmap items, not features
currently provided by the prototype.

---

## Repository structure

```text
.
├── app/                         Active Go monitor and Prometheus exporter
│   ├── exporter/                Prometheus metric collector and tests
│   ├── monitor/                 NVIDIA and Slurm/cgroup data sources
│   └── stress_test/             Slurm workload for cluster testing
├── deploy/                      Prometheus and Grafana (Docker Compose)
├── Documentation/               Implementation notes and draft architecture
├── tests/                       Root-level Python tests (currently a placeholder)
└── _app [deprecated]/           Previous monitor and database uploader
```

The active application is under `app/` and is a Go module. The root directory
does not define a Makefile or application runtime.

## Getting started

### Requirements

- Go version declared by `app/go.mod` (`1.27.2`).
- NVIDIA drivers and NVML for GPU metrics.
- Linux Slurm compute-node access and readable cgroup v2 files for job metrics.
- The `scontrol` command for retrieving Slurm job metadata.
- `make`, `goimports`, and `golangci-lint` for the corresponding Makefile targets.

The monitor's cgroup path is currently hard-coded in `app/monitor/cgroups_monitor.go`.
This prototype is intended to run on a configured Slurm node, not as a general
purpose local monitor.

### Build and run

```bash
cd app
make build
./slurm-cgroup-monitor <node-name> <port>
```

The HTTP metrics endpoint is available at `http://<node>:<port>/metrics`; the
Makefile's `make run` target uses `gpunode3` and port `9840`. To store the
metrics and view them in Grafana, see [the deployment guide](deploy/README.md).

### Tests and formatting

```bash
cd app
make test
make fmt
```

`make` with no target runs formatting and linting. For the root Python placeholder
test, run `python -m pytest` from the repository root after installing
`requirements.txt`. The placeholder is not application coverage.

See [the active app guide](app/README.md), [implementation notes](Documentation/DOCUMENTATION.md),
and [the documentation index](Documentation/README.md) for details.

---

## Contributing

### Branch strategy

```
main    <- stable, production-ready code only. Never push directly.
dev     <- active development. All feature branches merge here first.
```

All changes go through a pull request. PRs to `main` require 1 approval and passing CI. PRs to `dev` require 1 approval.

### Workflow

```bash
# Start from an up-to-date dev branch
git checkout dev
git pull origin dev

# Create a feature branch
git checkout -b feat/estimator-sacct-query

# Make changes, commit using Conventional Commits (see below)
git commit -m "feat(estimator): add sacct query with 90-day window"

# Push and open a PR targeting dev
git push origin feat/estimator-sacct-query
```

### Commit message format — Conventional Commits

Every commit must follow this format:

```
<type>(<scope>): <description>
```

**Rules:**

- Type and scope must be lowercase
- Description must be lowercase and start with a verb
- Description max 72 characters
- No period at the end

#### Types

| Type       | When to use                                          |
| ---------- | ---------------------------------------------------- |
| `feat`     | A new feature or functionality                       |
| `fix`      | A bug fix                                            |
| `docs`     | Documentation changes only                           |
| `refactor` | Code restructuring without changing behaviour        |
| `test`     | Adding or updating tests                             |
| `chore`    | Build process, CI, dependencies — no production code |
| `perf`     | Performance improvements                             |

#### Allowed scopes

| Scope       | What it covers                                    |
| ----------- | ------------------------------------------------- |
| `estimator` | `eden-estimate` CLI and sacct querying logic      |
| `monitor`   | In-job GPU/CPU metric collection                  |
| `storage`   | SQLite writers, rsync sync, database schema       |
| `analysis`  | Phase 0 notebooks and data exploration            |
| `infra`     | Prometheus, Grafana, PostgreSQL, systemd services |
| `docs`      | Documentation, ADRs, admin proposal               |
| `ci`        | GitHub Actions workflows, commitlint config       |

#### Examples

```bash
# ✅ Correct
feat(estimator): add sacct query with 90-day window
fix(monitor): handle missing gpu_util when nvidia-smi returns null
docs(decisions): add ADR-002 for pynvml over gpustat
refactor(storage): replace log file writer with sqlite3 insert
test(estimator): add unit tests for median resource calculation
chore(ci): add pytest workflow for pull requests
perf(monitor): reduce nvidia-smi subprocess calls using pynvml directly

# ❌ Wrong — scope not on the allowed list
feat(database): add connection pool

# ❌ Wrong — description starts with uppercase
fix(monitor): Handle missing gpu_util

# ❌ Wrong — missing scope
feat: add resource estimator

# ❌ Wrong — unknown type
update(estimator): improve query performance
```

### What not to commit

The `.gitignore` excludes the following — never force-add these:

- `*.csv` — sacct exports may contain usernames
- `*.db` — local SQLite databases
- `*.log` — raw metric logs
- `.env` — credentials and configuration secrets

---

## Documentation

- [Documentation index](Documentation/README.md)
- [Current implementation](Documentation/DOCUMENTATION.md)
- [Draft exporter architecture](Documentation/eden-job-exporter-spec.md)
- [Older project notes](Documentation/Prace.md)

The architecture specification describes a target design; it is not a guarantee
that each proposed feature exists in the active application.
