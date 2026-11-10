#include <omp.h>
int main() {
    #pragma omp parallel num_threads(64)
    {
        while(1); // 64 wątki palące CPU pod jednym PID-em
    }
}