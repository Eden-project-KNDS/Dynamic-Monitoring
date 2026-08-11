# Send logs to the database

This program reads CPU and GPU metric log files for a Slurm job and uploads them into a PostgreSQL database.

## Prerequisites

- A PostgreSQL server must already be running.
- The target database must already exist before running the program.
- The database must have TimescaleDB installed and enabled, because the program creates hypertables for `cpumetric` and `gpumetric`.
- The connection settings in [main.go](main.go) must be updated for your environment. In particular, change the host/IP address in the connection string from `localhost` if your database is on another machine.
- The program expects the log files to be present in the current working directory:
  - `usage_cpu_ram_<job-id>.log`
  - `usage_gpu_<job-id>.log`

## Build

From this folder, run:

```bash
make
```

This creates the binary `eden-savelog`.

## Run

Run the program with the Slurm job ID and account name:

```bash
./eden-savelog --job-id <slurm-job-id> --account <account-name>
```

Example:

```bash
./eden-savelog --job-id 12345 --account myproject
```

## What it creates

The program initializes these tables if they do not already exist:

- `useraccount` — stores `job_id` and `account`
- `cpumetric` — stores CPU log rows, with a hypertable on `time`
- `gpumetric` — stores GPU log rows, with a hypertable on `time`

## Notes

- The connection string is hard-coded in [main.go](main.go). Update `host`, `port`, `user`, `password`, and `dbname` as needed.
- If your database is remote, replace `localhost` with the correct host or IP address.
- The program currently does not remove log files after saving, because the removal code is commented out in `main.go`.
- If TimescaleDB is not available, the `create_hypertable` calls will fail.
  