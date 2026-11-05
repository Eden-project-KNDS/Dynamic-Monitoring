package monitor

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// Benchmark symulujący czytanie cgroup.procs z 10 000 procesów-dzieci
func BenchmarkGetAllPIDsFast(b *testing.B) {
	// 1. Przygotowanie środowiska testowego (nie wlicza się do czasu testu)
	tmpDir := b.TempDir()
	cgroupFile := filepath.Join(tmpDir, "cgroup.procs")

	f, err := os.Create(cgroupFile)
	if err != nil {
		b.Fatal(err)
	}

	// Generujemy 10 000 fikcyjnych PID-ów
	for i := 1; i <= 10000; i++ {
		f.WriteString(strconv.Itoa(i) + "\n")
	}
	f.Close()

	// 2. Resetujemy stoper - mierzymy tylko i wyłącznie Twój algorytm
	b.ResetTimer()

	// 3. Właściwa pętla benchmarka (Go odpali ją tysiące razy)
	for i := 0; i < b.N; i++ {
		// Zamień 'GetAllPIDsFast' na nazwę swojej funkcji przyjmującej ścieżkę do pliku
		v := getAllPidsFromDir("1", tmpDir)
		if v == nil {
			b.Fatal("pids empty\n")
		}

	}
}
