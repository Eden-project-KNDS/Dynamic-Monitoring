#include <iostream>
#include <vector>
#include <chrono>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>
#include <cuda_runtime.h>
#include <cmath>

// Ciężki kernel GPU wykonujący operacje trygonometryczne na zaalokowanej pamięci
__global__ void heavy_gpu_math(float* data, size_t n) {
    size_t idx = blockIdx.x * blockDim.x + threadIdx.x;
    if (idx < n) {
        float v = data[idx];
        #pragma unroll 10
        for (int i = 0; i < 100; ++i) {
            v = sinf(v) * cosf(v) + tanf(v);
        }
        data[idx] = v;
    }
}

// Funkcja paląca cykle procesora (CPU)
void stress_cpu() {
    volatile double d = 1.0;
    for(int i = 1; i < 50000; ++i) {
        d += sin((double)i) * cos((double)i);
    }
}

// Główna pętla obciążeniowa dla każdego procesu
void run_workload(int duration, size_t mem_bytes) {
    pid_t pid = getpid();
    
    // Alokacja pamięci VRAM dla tego konkretnego procesu
    float* d_data = nullptr;
    cudaError_t err = cudaMalloc(&d_data, mem_bytes);
    if (err != cudaSuccess) {
        std::cerr << "[PID " << pid << "] Błąd cudaMalloc: " << cudaGetErrorString(err) << std::endl;
        exit(1);
    }
    
    size_t num_elements = mem_bytes / sizeof(float);
    cudaMemset(d_data, 1, mem_bytes);
    
    int threads_per_block = 256;
    int blocks = (num_elements + threads_per_block - 1) / threads_per_block;
    // Ograniczenie liczby bloków w jednym wywołaniu, by nie wywołać GPU Timeout (TDR)
    if (blocks > 100000) blocks = 100000;

    auto start_time = std::chrono::steady_clock::now();
    
    std::cout << "[PID " << pid << "] Rozpoczęto obciążenie. Zaalokowano " 
              << mem_bytes / (1024 * 1024) << " MB VRAM." << std::endl;

    while(true) {
        auto now = std::chrono::steady_clock::now();
        if (std::chrono::duration_cast<std::chrono::seconds>(now - start_time).count() >= duration) {
            break;
        }
        
        // 1. Wysyłamy asynchroniczne zadanie na GPU
        heavy_gpu_math<<<blocks, threads_per_block>>>(d_data, num_elements);
        
        // 2. W czasie gdy GPU liczy, obciążamy rdzeń procesora na 100%
        for(int i = 0; i < 100; ++i) {
            stress_cpu();
        }
        
        // 3. Czekamy na koniec iteracji na GPU przed kolejnym cyklem
        cudaDeviceSynchronize();
    }
    
    cudaFree(d_data);
    std::cout << "[PID " << pid << "] Zakończono pracę." << std::endl;
}

int main(int argc, char** argv) {
    int duration = 120; // Domyślnie 2 minuty
    int num_procs = 8;  // Liczba procesów "dzieci" (łącznie z rodzicem)
    size_t total_mem = 8ULL * 1024 * 1024 * 1024; // 8 GB VRAM (łącznie)
    
    if (argc > 1) duration = std::atoi(argv[1]);
    if (argc > 2) num_procs = std::atoi(argv[2]);
    
    size_t mem_per_proc = total_mem / num_procs;
    
    std::cout << "=== START STRESS TESTU ===" << std::endl;
    std::cout << "Czas: " << duration << "s | Procesy: " << num_procs 
              << " | RAM GPU na proces: " << mem_per_proc / (1024 * 1024) << " MB" << std::endl;
    
    std::vector<pid_t> children;
    
    // Rozwidlanie procesów
    for (int i = 0; i < num_procs - 1; ++i) {
        pid_t pid = fork();
        if (pid == 0) {
            // Kod dziecka: inicjuje własny kontekst CUDA i obciąża system
            run_workload(duration, mem_per_proc);
            exit(0);
        } else if (pid > 0) {
            children.push_back(pid);
        } else {
            std::cerr << "Błąd fork()!" << std::endl;
        }
    }
    
    // Rodzic również wykonuje pracę jako jeden z 'num_procs'
    run_workload(duration, mem_per_proc);
    
    // Rodzic czeka na zakończenie wszystkich procesów potomnych
    for (pid_t child : children) {
        waitpid(child, NULL, 0);
    }
    
    std::cout << "=== WSZYSTKIE PROCESY ZAKOŃCZONE ===" << std::endl;
    return 0;
}