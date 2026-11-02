.PHONY: default fmt lint test run build test-job test-metrics clean

BIN=slurm-cgroup-monitor

# Domyślny cel (wywoływany przez wpisanie samo 'make')
default: fmt lint

# Formatuje kod Go przy użyciu gofmt i goimports
fmt:
	go fmt ./...	
	goimports -w .

# Uruchamia linter do sprawdzania jakości i błędów w kodzie
lint:
	golangci-lint run

# Uruchamia wszystkie testy jednostkowe
test:
	go test -v ./...

run: build
	./$(BIN) gpunode3
# Kompiluje demona monitorującego zasoby Slurma (z wyciszeniem ostrzeżeń CGO)
build:
	CGO_CFLAGS="-Wno-deprecated-declarations" go build -o $(BIN) .
# --- CELE DO TESTOWANIA PROMTHEUSA I SLURMA ---
# 1. Wysyła testowe obciążenie do Slurma (2 rdzenie na 5 minut, 100% użycia CPU)
# Uruchamiaj to z poziomu katalogu, w którym masz prawo zapisu (nie z /sys)!
test-job:
	@echo "Wrzucam sztuczne zadanie obciążające do Slurma..."
	sbatch -c 2 --wrap="timeout 300 sh -c 'yes > /dev/null & yes > /dev/null & wait'"

# 2. Odpytuje endpoint Prometheusa i filtruje nasze metryki
test-metrics:
	@echo "Pobieram metryki 'eden_' z localhost:9840/metrics..."
	curl -sS localhost:9840/metrics | grep -E '^eden_(job|gpu)' || (echo "\n[!] Nie znaleziono metryk. Czy slurm-cgroup-monitor jest włączony i czy zadanie działa?" && exit 1)

clean:
	rm -f *.out $(BIN) 