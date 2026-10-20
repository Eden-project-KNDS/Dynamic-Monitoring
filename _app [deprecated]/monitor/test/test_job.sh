#!/bin/bash

# 1. Simulate Slurm setting the Job ID environment variable
export SLURM_JOB_ID="test_98765"
echo "Starting mock job: $SLURM_JOB_ID"

# 2. Start the monitor in the background (using our compiled binary)
./../eden-monitor --job-id $SLURM_JOB_ID &
MONITOR_PID=$!

echo "Monitor started with PID: $MONITOR_PID"

# 3. Simulate the actual training process (e.g., python train.py)
echo "Simulating neural network training for 5 seconds..."
sleep 5

# 4. Kill the monitor (This tests the signal handling in Go)
echo "Training finished. Sending SIGTERM to monitor..."
kill $MONITOR_PID

# Wait a moment for Go to clean up its child processes safely
wait $MONITOR_PID 
echo "Job script finished completely!"