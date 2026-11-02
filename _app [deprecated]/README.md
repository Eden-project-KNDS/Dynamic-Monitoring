# Deprecated Application

This directory contains the previous monitoring implementation. It is kept
for historical reference and is not the active application; use the Go module
in [`app/`](../app/README.md) for current development.

The legacy project has two separate tools:

- [`monitor/`](monitor/README.md) runs `pidstat` and `nvidia-smi` to collect
  process-oriented CPU, memory, and GPU logs.
- [`send_logs_to_db/`](send_logs_to_db/README.md) reads those logs and uploads
  them to PostgreSQL/TimescaleDB.

These tools have their own setup and configuration requirements. Their log
format and database pipeline are separate from the active Prometheus exporter.