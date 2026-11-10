#!/bin/bash
#SBATCH --job-name=race_conditions
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=2
#SBATCH --time=00:10:00



# Uruchamia 500 bardzo krótkich procesów w szybkim tempie
for i in {1..2_000_000}; do
    python3 -c "import math, sys; print('quick')"  &
done