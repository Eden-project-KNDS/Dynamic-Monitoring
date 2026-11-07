#!/bin/bash
#SBATCH --job-name=stress_gpu
#SBATCH --nodes=1
#SBATCH --gpus=1
#SBATCH --time=00:10:00

# Aktywuj środowisko z PyTorchem (dostosuj ścieżkę do swojego klastra)
module load python/3.10
module load pytorch

python3 stress_gpu.py