# Prometheus and Grafana

This directory runs the storage and visualization side of the monitor:
Prometheus scrapes `slurm-cgroup-monitor` on a node and stores the metrics,
Grafana shows them on the **Eden jobs** dashboard.

```text
node: slurm-cgroup-monitor :9840/metrics
        │  every 15 s (scrape)
        ▼
Prometheus :9090  (time-series database, 30-day retention)
        │  PromQL
        ▼
Grafana :3000     (Eden jobs dashboard)
```

| File | Purpose |
|---|---|
| `docker-compose.yml` | Prometheus and Grafana containers, volumes, ports |
| `prometheus/prometheus.yml` | what Prometheus scrapes and how often |
| `grafana/provisioning/datasources/prometheus.yml` | Grafana's Prometheus data source |
| `grafana/provisioning/dashboards/eden.yml` | loads dashboards from `grafana/dashboards/` |
| `grafana/dashboards/eden-jobs.json` | the Eden jobs dashboard |
| `example.env` | template for `.env` (Grafana admin password) |

## Requirements

- Docker with the Compose plugin (`docker compose version`).
- A running `slurm-cgroup-monitor` (see [the app guide](../app/README.md)).
- SSH access to the node. On the MiNI network the nodes are reached through the
  jump host `ssh.mini.pw.edu.pl`.

## SSH setup (once, on your computer)

Add to `~/.ssh/config`:

```text
Host gpunode3
    HostName gpunode3
    User <your-login>
    ProxyJump <your-login>@ssh.mini.pw.edu.pl
```

Afterwards `ssh gpunode3` logs in through the jump host, and the tunnels below
work with the short name.

## Option A: run on the node (recommended)

The stack keeps collecting data when your laptop is closed.

1. Start the monitor so that it survives logging out:

   ```bash
   tmux new -s exporter
   cd ~/Dynamic-Monitoring/app && make build
   ./slurm-cgroup-monitor gpunode3 9840
   ```

   Detach with `Ctrl+B`, then `D`. Reattach with `tmux attach -t exporter`.

2. Set the Grafana admin password **before the first start** (Grafana reads it
   only when it creates its database):

   ```bash
   cd ~/Dynamic-Monitoring/deploy
   cp example.env .env
   nano .env            # set GRAFANA_ADMIN_PASSWORD, e.g. from: openssl rand -base64 18
   chmod 600 .env
   ```

3. Start Prometheus and Grafana (use `sudo` if your user is not in the `docker`
   group):

   ```bash
   sudo docker compose up -d
   sudo docker compose ps
   ```

4. Check that Prometheus scrapes the monitor (both targets should be `up`):

   ```bash
   curl -s localhost:9090/api/v1/targets | grep -o '"health":"[a-z]*"'
   ```

5. On your computer, open tunnels and keep the window open:

   ```bash
   ssh -N -L 3000:localhost:3000 -L 9090:localhost:9090 gpunode3
   ```

   Grafana: <http://localhost:3000> (user `admin`, password from `.env`),
   Prometheus: <http://localhost:9090>.

The containers restart automatically (`restart: unless-stopped`). The monitor in
`tmux` does not: after a node reboot, start it again (step 1).

## Option B: run on your computer against a node

Useful for checking changes to the dashboard or Prometheus config without
touching the node. Data is collected only while the tunnel is open.

1. On the node, start the monitor as in option A, step 1.
2. On your computer, tunnel the monitor's port:

   ```bash
   ssh -N -L 9840:localhost:9840 gpunode3
   curl -s localhost:9840/metrics | head    # should print eden_* metrics
   ```

3. Start the stack:

   ```bash
   cd deploy
   cp example.env .env                       # any password is fine locally
   docker compose up -d
   ```

Prometheus reaches the tunnel through `host.docker.internal:9840`; the same
`prometheus.yml` works in both options.

## Checking that it works

1. **Prometheus → Status → Target health**: `eden` and `prometheus` are UP.
2. **Grafana → Dashboards → Eden → Eden jobs**.
3. Generate load on the node and watch the dashboard (allow about a minute):

   ```bash
   cd ~/Dynamic-Monitoring/app && make test-job            # 2 CPUs busy for 5 min
   cd ~/Dynamic-Monitoring/app/stress_test && make run     # 2 GiB RAM, 1 CPU for 3 min
   ```

   Submit jobs from a writable directory (not from `/sys`), and pass `--mem`
   when submitting your own jobs: on gpunode3 a job without `--mem` reserves all
   node memory and blocks other jobs.

4. Useful queries in Prometheus (**Query** page):

   | Query | Meaning |
   |---|---|
   | `up` | 1 = target scraped successfully |
   | `rate(eden_job_cpu_usage_seconds_total[1m])` | CPU cores used per job |
   | `rate(eden_job_cpu_usage_seconds_total[1m]) / eden_job_cpus_allocated` | CPU efficiency (0–1) |
   | `max_over_time(eden_job_memory_used_bytes[1h])` | peak memory per job (kernels without `memory.peak`) |
   | `eden_gpu_utilization_ratio * on(job_id) group_left(user) eden_job_info` | GPU utilization with the job's user |

## Configuration

- **Scrape interval and timeout**: `prometheus/prometheus.yml` (15 s / 10 s).
  The monitor runs job discovery during each scrape, so keep the timeout well
  above typical `scontrol` latency.
- **Retention**: `--storage.tsdb.retention.time` in `docker-compose.yml` (30 days).
- **Ports** are bound to `127.0.0.1`, so Prometheus and Grafana are reachable
  only through an SSH tunnel. Docker bypasses host firewalls such as `ufw`, so
  keep the `127.0.0.1:` prefix on shared machines.
- **Data** lives in the Docker volumes `eden-monitoring_prometheus-data` and
  `eden-monitoring_grafana-data`. `docker compose down` keeps them;
  `docker compose down -v` deletes all collected history.
- **Grafana password**: changing `.env` has no effect after the first start. Use
  the Grafana UI or
  `docker compose exec grafana grafana cli admin reset-admin-password <new>`.

## Editing the dashboard

The dashboard file is the source of truth (`allowUiUpdates: false`), so changes
saved in the UI are not kept. Edit the dashboard in Grafana, then use
**Export → Export as JSON**, save it as `grafana/dashboards/eden-jobs.json` and
commit it. Grafana reloads the file within about 10 seconds.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| `permission denied ... /var/run/docker.sock` | your user cannot use Docker; prefix commands with `sudo` |
| `Cannot connect to the Docker daemon` (macOS) | Docker Desktop is not running; start it (`open -a Docker`) |
| target `eden` DOWN, `connection refused` | the monitor or the SSH tunnel is not running |
| `bind: address already in use` from the monitor | another monitor already uses the port (check `tmux ls`) |
| tunnel fails: port 3000 or 9090 in use | a local stack is running; stop it with `docker compose down` |
| job missing on the dashboard | the job is pending (`squeue`) or failed; read `slurm-<id>.out` |
| GPU `job_id` empty | the job has no process on the card (an idle reserved GPU is not detected yet) |

## Known limitations

- The monitor runs in `tmux`, not as a systemd service, so it does not start
  after a reboot.
- Prometheus on the node starts with empty history.
- gpunode3 runs kernel 5.15, which has no `memory.peak`; use `max_over_time`.
- gpunode3's Slurm does not set `memory.max`, so
  `eden_job_memory_allocated_bytes` is absent there.
- GPUs are attributed to jobs through processes running on the card, so a
  reserved but idle GPU shows an empty `job_id`.
