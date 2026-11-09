#include <stdio.h>
#include <stdlib.h>
#include <time.h>
#include <math.h>

// Ustawienia testu
#define GIGABYTES 2 
#define DURATION_SECONDS 180 

int main() {
    printf("Rozpoczynam test obciążeniowy...\n");
    
    // 1. OBCIĄŻENIE RAM
    size_t num_bytes = (size_t)GIGABYTES * 1024 * 1024 * 1024;
    size_t num_elements = num_bytes / sizeof(double);
    
    printf("Alokowanie %d GB RAM...\n", GIGABYTES);
    double *memory_hog = (double *)malloc(num_bytes);
    if (memory_hog == NULL) {
        fprintf(stderr, "Błąd: Nie udało się zaalokować pamięci!\n");
        return 1;
    }

    // Zapis do pamięci jest kluczowy dla cgroups.
    // Wymusza przydzielenie fizycznych stron pamięci (Physical RAM mapping).
    printf("Wypełnianie RAMu danymi (wymuszanie alokacji fizycznej)...\n");
    for (size_t i = 0; i < num_elements; i++) {
        memory_hog[i] = (double)i * 3.14159;
    }
    printf("RAM zajęty pomyślnie.\n");

    // 2. OBCIĄŻENIE CPU
    printf("Rozpoczynam 'smażenie' procesora na %d sekund...\n", DURATION_SECONDS);
    time_t start_time = time(NULL);
    
    double dummy_calc = 0.0;
    
    // Zewnętrzna pętla kontroluje czas działania (3 minuty)
    while ((time(NULL) - start_time) < DURATION_SECONDS) {
        // Wewnętrzna pętla z operacjami trygonometrycznymi obciąża ALU (Arithmetic Logic Unit)
        for (int i = 1; i < 100000; i++) {
            dummy_calc += sin((double)i) * tan((double)i) / sqrt((double)i);
        }
    }

    // Wypisanie wyniku jest konieczne, aby kompilator nie usunął pętli podczas optymalizacji (np. -O2 lub -O3)
    printf("Czas upłynął. Wynik operacji śmieciowych: %f\n", dummy_calc);
    
    // Sprzątanie
    free(memory_hog);
    printf("Pamięć zwolniona. Zakończenie pracy.\n");

    return 0;
}