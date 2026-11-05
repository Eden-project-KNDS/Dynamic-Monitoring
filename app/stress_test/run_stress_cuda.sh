#!/bin/bash
#SBATCH --job-name=cuda_stress_test
#SBATCH --output=stress_%j.out
#SBATCH --error=stress_%j.err
#SBATCH --time=00:05:00
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=2
#SBATCH --gres=gpu:1

# Ustawienie opcji zatrzymania w przypadku błędu
set -e

echo "=== Uruchamianie na węźle: $(hostname) ==="
echo "=== Przydzielone GPU (CUDA_VISIBLE_DEVICES): $CUDA_VISIBLE_DEVICES ==="

# 1. Kompilacja (jeśli plik wykonywalny nie istnieje)
if [ ! -f ./stress_gpu ]; then
    echo "Kompilacja stress_gpu.cu przy użyciu nvcc..."
    nvcc -O3 stress_gpu.cu -o stress_gpu
fi

# 2. Uruchomienie stres-testu na 120 sekund
# (daje wystarczająco dużo czasu na wywoływanie 15-sekundowego interwału demona)
./stress_gpu 120