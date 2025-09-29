.PHONY: all test bench bench-mem bench-cpu bench-compare fmt clean help

# Default target
all:
	go test ./...
	go test ./... -short -race
	go vet

help:
	@echo "Available targets:"
	@echo "  all          - Run tests, race detector, and vet (default)"
	@echo "  test         - Run all tests"
	@echo "  bench        - Run all benchmarks"
	@echo "  bench-mem    - Run benchmarks with memory profiling"
	@echo "  bench-cpu    - Run benchmarks with CPU profiling"
	@echo "  bench-compare - Compare benchmark runs"
	@echo "  fmt          - Format code"
	@echo "  clean        - Clean build artifacts"

# Run tests
test:
	go test -v ./...

# Run all benchmarks
bench:
	go test -bench=. -benchmem -benchtime=10x

# Run benchmarks with memory profiling
bench-mem:
	go test -bench=. -benchmem -benchtime=10x -memprofile=mem.prof
	@echo "View memory profile with: go tool pprof mem.prof"

# Run benchmarks with CPU profiling
bench-cpu:
	go test -bench=. -benchtime=10x -cpuprofile=cpu.prof
	@echo "View CPU profile with: go tool pprof cpu.prof"

# Compare benchmarks using benchstat
bench-compare:
	@echo "Running baseline benchmarks..."
	go test -bench=. -benchmem -benchtime=10x -count=5 > bench-baseline.txt
	@echo "Apply your changes, then run 'make bench-new' to generate new results"

bench-new:
	@echo "Running new benchmarks..."
	go test -bench=. -benchmem -benchtime=10x -count=5 > bench-new.txt
	@echo "Comparing results..."
	@if command -v benchstat > /dev/null 2>&1; then \
		benchstat bench-baseline.txt bench-new.txt; \
	else \
		echo "benchstat not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

# Format code
fmt:
	gofmt -w .
	go mod tidy

# Clean build artifacts
clean:
	rm -f *.prof bench-*.txt
	go clean -testcache