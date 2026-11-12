#!/bin/bash
#SBATCH --job-name=stress_pids
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=2
#SBATCH --time=00:10:00

echo "Rozpoczynam generowanie 2000 procesów-dzieci w tle..."

# Uruchamiamy 2000 uśpionych procesów.
# Z perspektywy cgroups, plik cgroup.procs nagle spuchnie do 2000 linii.
for i in {1..2000}; do
    sleep 300 &
done

echo "Procesy utworzone. Czekam na ich zakończenie..."
# wait blokuje główny skrypt, dopóki wszystkie zadania w tle (sleep) się nie skończą
wait 

echo "Zadanie zakończone."