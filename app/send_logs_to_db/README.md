# Send logs to the database

This program reads CPU and GPU metric log files for a Slurm job and uploads them into a PostgreSQL database.

## Prerequisites

- A PostgreSQL server must already be running.
- The target database must already exist before running the program.
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

## Important notes

- The program uses the connection string in [main.go](main.go). Update `host`, `port`, `user`, `password`, and `dbname` as needed.
- In the future it will be replace with .env file
- If the database is remote, replace `localhost` with the correct IP address or hostname.
- The program will create the `resource_metrics` table and a hypertable if they do not already exist.
- After saving data to database program removes log files
  