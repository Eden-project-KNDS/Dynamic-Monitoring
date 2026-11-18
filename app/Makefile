.PHONY: default fmt lint test run build test-job test-metrics clean

BIN=slurm-monitor
LISTEN_ADDRESS=9840
SERVICE_FILE=config/slurm-monitor.service
CONFIG_FILE=config/config.env 

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

benchmark:
	go test -bench=. -benchmem ./monitor

# Kompiluje demona monitorującego zasoby Slurma (z wyciszeniem ostrzeżeń CGO)
build:
	CGO_CFLAGS="-Wno-deprecated-declarations" go build -o $(BIN) .

daemon_status:
	systemctl status $(BIN)


# setting up linux daemon
set_daemon: build
	cp $(BIN) /usr/local/bin/
	chown root:root /usr/local/bin/$(BIN)
	chmod 755 /usr/local/bin/$(BIN)
	mkdir -p /etc/$(BIN)
	cp $(SERVICE_FILE) /etc/systemd/system/
	cp $(CONFIG_FILE) /etc/$(BIN)/
	systemctl daemon-reload
	systemctl enable $(BIN)
	systemctl start $(BIN)
	$(MAKE) daemon_status

clean:
	rm -f *.out $(BIN) 