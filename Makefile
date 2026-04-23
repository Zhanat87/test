.PHONY: test test-race test-concurrent

test:
	go test ./...

test-race:
	go test ./... -race

test-concurrent:
	go test ./... -race -run '^(TestReserve_ConcurrentOversell|TestReserveMultiple_Atomicity)$$'

