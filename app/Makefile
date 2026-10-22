.PHONY: default fmt lint test build

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

# Kompiluje demona monitorującego zasoby Slurma (z wyciszeniem ostrzeżeń CGO)
build:
	CGO_CFLAGS="-Wno-deprecated-declarations" go build -o slurm-cgroup-monitor .