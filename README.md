# Eden Cluster Monitor

A monitoring and resource optimization system for the GPU cluster at the Faculty of Mathematics and Information Science (MiNI), Warsaw University of Technology.

---

## Problem

Research teams and students using the Eden cluster (32× A100, 8× H100, 4× P100) routinely overprovision resources — reserving 72-hour time limits for jobs that finish in 4 hours, or allocating 4 GPUs when only 1 is used. This blocks resources for other users without any real need.

The existing monitoring solution generates thousands of log files per month and provides no tooling to help users make better decisions before submitting a job.

---

## What we're building

| Phase | Component             | Description                                                     |
| ----- | --------------------- | --------------------------------------------------------------- |
| 0     | Historical analysis   | `sacct` data analysis to quantify the problem                   |
| 1     | `eden-estimate` CLI   | Suggests realistic `#SBATCH` parameters based on job history    |
| 2     | Job efficiency report | Post-job report scoring GPU utilisation, memory, and time usage |
| 3     | Infrastructure        | SQLite → PostgreSQL, Prometheus, Grafana dashboard              |
| 4     | Web interface         | FastAPI backend + resource estimator form                       |

---

## Repository structure (TODO)

---

## Getting started (TODO)

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

## Documentation (TODO: partially done)

---
