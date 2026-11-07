#include <iostream>
#include <chrono>
#include <cstdlib>
#include <unistd.h>
#include <cuda_runtime.h>

// Kernel wykonujący intensywne obliczenia FMA (Fused Multiply-Add)
__global__ void stress_kernel(float* data, size_t n, int inner_iterations) {
    size_t idx = blockIdx.x * blockDim.x + threadIdx.x;
    if (idx < n) {
        float val = data[idx];
        #pragma unroll 8
        for (int i = 0; i < inner_iterations; ++i) {
            val = fmaf(val, 1.000001f, 0.000001f);
        }
        data[idx] = val;
    }
}

int main(int argc, char** argv) {
    int duration_sec = 60; // Domyślny czas trwania: 60 sekund
    if (argc > 1) {
        duration_sec = std::atoi(argv[1]);
    }

    int device_id = 0;
    cudaGetDevice(&device_id);

    pid_t pid = getpid();
    std::cout << "[STRESS TEST] Start na GPU ID: " << device_id 
              << " | PID procesu: " << pid 
              << " | Czas: " << duration_sec << "s" << std::endl;

    // Alokacja 2 GB VRAM na GPU
    size_t n = 512 * 1024 * 1024; // 512M elementów float = 2 GB
    size_t bytes = n * sizeof(float);

    float* d_data = nullptr;
    cudaError_t err = cudaMalloc(&d_data, bytes);
    if (err != cudaSuccess) {
        std::cerr << "Błąd cudaMalloc: " << cudaGetErrorString(err) << std::endl;
        return 1;
    }

    cudaMemset(d_data, 1, bytes);

    int threads_per_block = 256;
    int blocks = (n + threads_per_block - 1) / threads_per_block;

    auto start_time = std::chrono::steady_clock::now();

    while (true) {
        auto current_time = std::chrono::steady_clock::now();
        auto elapsed = std::chrono::duration_cast<std::chrono::seconds>(current_time - start_time).count();
        if (elapsed >= duration_sec) {
            break;
        }

        stress_kernel<<<blocks, threads_per_block>>>(d_data, n, 2000);
        cudaDeviceSynchronize();
    }

    cudaFree(d_data);
    std::cout << "[STRESS TEST] Zakończono pomyślnie na GPU " << device_id 
              << " (PID: " << pid << ")" << std::endl;

    return 0;
}