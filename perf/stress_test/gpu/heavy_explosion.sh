#!/bin/bash
#SBATCH --job-name=gpu_pid_explosion
#SBATCH --output=explosion_%j.out
#SBATCH --error=explosion_%j.err
#SBATCH --time=00:05:00
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=8
#SBATCH --gres=gpu:1

set -e

echo "=== Kompilacja heavy_explosion.cu ==="
if [ ! -f ./heavy_explosion ]; then
    nvcc -O3 heavy_explosion.cu -o heavy_explosion
fi

echo "=== Uruchamianie testu na węźle: $(hostname) ==="
# Parametry: <Czas_w_sekundach> <Liczba_procesów>
# Uruchamia na 120 sekund w 8 osobnych procesach, każdy z własnym PID
./heavy_explosion 120 8