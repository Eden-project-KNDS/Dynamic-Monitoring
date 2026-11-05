import torch
import time
import sys

if not torch.cuda.is_available():
    print("Błąd: CUDA niedostępna na tym węźle!")
    sys.exit(1)

print("Karta wykryta:", torch.cuda.get_device_name(0))
print("Alokowanie ~4GB VRAM na GPU...")

# Tworzymy ogromne tensory - to wymusi twardą alokację pamięci VRAM
rozmiar = 10000
x = torch.randn(rozmiar, rozmiar, device="cuda")
y = torch.randn(rozmiar, rozmiar, device="cuda")

print("Rozpoczęcie obciążania rdzeni CUDA (GPU Utilization na 100%)...")
start_time = time.time()

# Obciążamy kartę przez 5 minut
while time.time() - start_time < 300:
    # Mnożenie macierzy (matmul) to najcięższa operacja dla Tensor Cores
    z = torch.matmul(x, y)

    # Synchronizacja wymusza fizyczne zakończenie obliczeń przez GPU w każdej pętli
    torch.cuda.synchronize()

print("Zakończono test obciążeniowy GPU.")
