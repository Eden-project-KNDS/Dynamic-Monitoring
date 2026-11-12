# Prometheus and Grafana

This directory runs the storage and visualization layer of the monitoring stack. Prometheus scrapes the Slurm monitor on each compute node and stores the resulting `eden_*` time series, while Grafana presents the Eden jobs dashboard.

```text
node: slurm-monitor :9840/metrics
        │  every 15 s (scrape)
        ▼
Prometheus :9090  (time-series database, 30-day retention)
        │  PromQL
        ▼
Grafana :3000     (Eden jobs dashboard)
```

| File | Purpose |
|---|---|
| `docker-compose.yml` | Prometheus and Grafana containers, ports, and volumes |
| `prometheus/prometheus.yml` | what Prometheus scrapes and how often |
| `grafana/provisioning/datasources/prometheus.yml` | Grafana data source wiring |
| `grafana/provisioning/dashboards/eden.yml` | dashboard provisioning |
| `grafana/dashboards/eden-jobs.json` | the Eden jobs dashboard |
| `example.env` | template for `.env` with Grafana admin password |

## Requirements

- Docker with the Compose plugin (`docker compose version`)
- A running `slurm-monitor` process on the target node (see [../app/README.md](../app/README.md))
- SSH access to the node, typically via a jump host such as `ssh.mini.pw.edu.pl`

## SSH setup

Add this to `~/.ssh/config`:

```text
Host gpunode3
    HostName gpunode3
    User <your-login>
    ProxyJump <your-login>@ssh.mini.pw.edu.pl
```

Afterwards, `ssh gpunode3` can reach the compute node through the jump host, and the port-forwarding examples below work with the short name.

## Option A: run on the node (recommended)

The stack keeps collecting data while your laptop is closed.

1. Launch the monitor in a persistent session:

   ```bash
   tmux new -s exporter
   cd ~/Dynamic-Monitoring/app && make build
   ./slurm-monitor gpunode3 9840
   ```

   Detach with `Ctrl+B` then `D`. Reattach later with `tmux attach -t exporter`.

2. Set the Grafana admin password before the first start:

   ```bash
   cd ~/Dynamic-Monitoring/deploy
   cp example.env .env
   nano .env
   chmod 600 .env
   ```

   Set `GRAFANA_ADMIN_PASSWORD` to a strong value (for example, `openssl rand -base64 18`).

3. Start Prometheus and Grafana:

   ```bash
   sudo docker compose up -d
   sudo docker compose ps
   ```

4. Confirm that Prometheus can reach the target:

   ```bash
   curl -s localhost:9090/api/v1/targets | grep -o '"health":"[a-z]*"'
   ```

5. Keep the SSH tunnel open from your laptop:

   ```bash
   ssh -N -L 3000:localhost:3000 -L 9090:localhost:9090 gpunode3
   ```

   Grafana is available at <http://localhost:3000> with the `admin` user and the password from `.env`. Prometheus is available at <http://localhost:9090>.

The Docker Compose stack restarts automatically with `restart: unless-stopped`. The monitor itself does not survive a node reboot, so start it again after a reboot.

## Option B: run locally against a node

Useful when testing dashboard or Prometheus changes without leaving the monitor running on the node.

1. Start the monitor on the node as above.
2. On your laptop, tunnel the monitor port:

   ```bash
   ssh -N -L 9840:localhost:9840 gpunode3
   curl -s localhost:9840/metrics | head
   ```

3. Start the stack locally:

   ```bash
   cd deploy
   cp example.env .env
   docker compose up -d
   ```

Prometheus reaches the tunnel through `host.docker.internal:9840`; the same `prometheus.yml` works in both deployment patterns.

## Confirming the setup

1. Check Prometheus target health: `eden` and `prometheus` should be `UP`.
2. Open Grafana and load the **Eden jobs** dashboard.
3. Generate load on the node and observe the dashboard after a minute or so:

   ```bash
   cd ~/Dynamic-Monitoring/app && make test-job
   cd ~/Dynamic-Monitoring/app/stress_test && make run
   ```

4. Useful Prometheus queries:

   | Query | Meaning |
   |---|---|
   | `up` | `1` means the target scraped successfully |
   | `rate(eden_job_cpu_usage_seconds_total[1m])` | CPU cores used per job |
   | `rate(eden_job_cpu_usage_seconds_total[1m]) / eden_job_cpus_allocated` | CPU efficiency, value `0..1` |
   | `max_over_time(eden_job_memory_used_bytes[1h])` | peak memory per job |
   | `eden_gpu_utilization_ratio * on(job_id) group_left(user) eden_job_info` | GPU utilization joined to job user |

## Configuration

- **Scrape interval and timeout**: defined in [prometheus/prometheus.yml](prometheus/prometheus.yml)
- **Retention**: `--storage.tsdb.retention.time` in [docker-compose.yml](docker-compose.yml)
- **Ports** are bound to `127.0.0.1` so access is tunneled through SSH
- **Persistent data** lives in Docker volumes created by Compose; `docker compose down` keeps them, while `docker compose down -v` clears history
- **Grafana password**: change `.env` before the first start; after creation, update the password through the Grafana UI or `docker compose exec grafana grafana cli admin reset-admin-password <new>`

## Editing the dashboard

The dashboard file under [grafana/dashboards/eden-jobs.json](grafana/dashboards/eden-jobs.json) is the source of truth. Edit it in Grafana, then export it as JSON and commit the updated file so it is kept in version control.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| `permission denied ... /var/run/docker.sock` | Your user cannot access Docker; prefix commands with `sudo` |
| `Cannot connect to the Docker daemon` (macOS) | Docker Desktop is not running |
| `target eden DOWN` or `connection refused` | The monitor or SSH tunnel is not running |
| `bind: address already in use` | Another monitor already uses the expected port |
| Port 3000 or 9090 already in use | Stop the local stack with `docker compose down` |
| Job missing from the dashboard | The job may still be pending or failed; inspect the Slurm job output |
| GPU `job_id` empty | The job has no running process on the GPU; this is a known limitation |

## Known limitations

- The monitor runs in `tmux` rather than a systemd service, so it does not start after a reboot unless restarted manually.
- Prometheus starts with empty history on a fresh node.
- Some kernels, including gpunode3's 5.15, do not expose `memory.peak`; use `max_over_time` in PromQL instead.
- GPUs are attributed to jobs by looking at processes currently running on the card, so an allocated but idle GPU may still show an empty `job_id`.