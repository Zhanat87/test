.PHONY: test test-race test-concurrent bench benchmem bench-compare bench-compare-mem

test:
	go test ./...

test-race:
	go test ./... -race

test-concurrent:
	go test ./... -race -run '^(TestReserve_ConcurrentOversell|TestReserveMultiple_Atomicity)$$'

bench:
	go test ./... -run '^$$' -bench . 

benchmem:
	go test ./... -run '^$$' -bench . -benchmem

bench-compare:
	go test ./... -run '^$$' -bench 'Benchmark(ReserveParallel|GetStockParallel)_' 

bench-compare-mem:
	go test ./... -run '^$$' -bench 'Benchmark(ReserveParallel|GetStockParallel)_' -benchmem

