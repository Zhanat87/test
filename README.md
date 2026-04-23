# Project Testing and Benchmarking

This Makefile provides convenient shortcuts for running tests, race detection, and benchmarks for the Go project, specifically focusing on concurrency and reserve logic.

## Commands

### Testing
*   `make test`: Runs all tests in the project.
*   `make test-race`: Runs all tests with the data race detector enabled.
*   `make test-concurrent`: Runs specific tests (`TestReserve_ConcurrentOversell`, `TestReserveMultiple_Atomicity`) to verify concurrent behavior.

### Benchmarking
*   `make bench`: Runs all benchmarks with memory statistics.
*   `make bench-mem`: Runs benchmarks with detailed memory allocation statistics.
*   `make bench-compare`: Runs benchmarks and compares them with previous results (`bench.old`).
*   `make bench-compare-mem`: Runs benchmarks with memory stats and compares with previous results.

### Specific Function Benchmarks
*   `make test-go`: Runs `BenchmarkReserveParallel` and `BenchmarkGetStockParallel` with memory statistics.
*   `make bench-compare-go`: Runs `BenchmarkReserveParallel` and `BenchmarkGetStockParallel` and compares with previous results.

## Usage
Run the commands in your terminal:
```bash
make test-race
make bench-compare
```
