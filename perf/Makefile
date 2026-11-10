.ONESHELL:
.PHONY: run_perf analyze

PID ?= 536851
TIMER ?= 60

run_perf:
	cd $(HOME)
	sudo perf record -F 99 -p $(PID) -g -- sleep $(TIMER)

analyze:
	cd $(HOME)
	sudo perf script > out.perf
	./FlameGraph/stackcollapse-perf.pl out.perf > out.folded
	./FlameGraph/flamegraph.pl out.folded > graph.svg
	mv graph.svg edenMonitor/Dynamic-Monitoring/perf/