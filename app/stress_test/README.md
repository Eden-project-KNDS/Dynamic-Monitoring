# Slurm Stress Workload

This directory contains a C program that allocates and touches approximately
2 GiB of memory, then performs CPU work for about three minutes. It is intended
to exercise resource monitoring on a Slurm compute node, not as a unit test.

## Requirements

- GCC and `make` to compile the workload.
- Access to a Slurm cluster and permission to submit jobs for `make run`.
- A node allocation with enough available memory for the workload.

## Build and submit

From this directory:

```bash
make
make run
```

`make run` submits the binary with `sbatch`, requesting one CPU, 4 GiB of
memory, and a five-minute time limit. The program itself runs for approximately
three minutes. This consumes cluster resources and may incur account or queue
usage; submit only where you are authorized to run test workloads.

To run the binary directly on an allocated node, use `./stress_test`. Clean up
the generated executable and output files with `make clean`.