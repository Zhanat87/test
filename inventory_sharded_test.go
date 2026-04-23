package inventory

import (
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type inventoryAPI interface {
	GetStock(productID string) int
	Reserve(productID string, quantity int) error
	ReserveMultiple(items []ReserveItem) error
}

func TestShardedReserve_ConcurrentOversell(t *testing.T) {
	svc := NewShardedInventoryService(32, map[string]*Product{
		"p1": {ID: "p1", Name: "Product 1", Stock: 100},
	})

	const goroutines = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)

	start := make(chan struct{})

	var success int32
	var failure int32

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			if err := svc.Reserve("p1", 1); err != nil {
				atomic.AddInt32(&failure, 1)
				return
			}
			atomic.AddInt32(&success, 1)
		}()
	}

	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&success); got != 100 {
		t.Fatalf("expected 100 successful reservations, got %d", got)
	}
	if got := atomic.LoadInt32(&failure); got != 100 {
		t.Fatalf("expected 100 failed reservations, got %d", got)
	}
	if stock := svc.GetStock("p1"); stock != 0 {
		t.Fatalf("expected final stock 0, got %d", stock)
	}
}

func TestShardedReserveMultiple_Atomicity(t *testing.T) {
	svc := NewShardedInventoryService(32, map[string]*Product{
		"a": {ID: "a", Name: "A", Stock: 10},
		"b": {ID: "b", Name: "B", Stock: 5},
	})

	err := svc.ReserveMultiple([]ReserveItem{
		{ProductID: "a", Quantity: 8},
		{ProductID: "b", Quantity: 8},
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if got := svc.GetStock("a"); got != 10 {
		t.Fatalf("expected stock of A unchanged (10), got %d", got)
	}
	if got := svc.GetStock("b"); got != 5 {
		t.Fatalf("expected stock of B unchanged (5), got %d", got)
	}
}

func benchmarkReserveParallel(b *testing.B, svc inventoryAPI, productIDs []string) {
	b.Helper()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(runtimeProcIDHint())))
		n := len(productIDs)
		for pb.Next() {
			id := productIDs[r.Intn(n)]
			_ = svc.Reserve(id, 1)
		}
	})
}

func benchmarkGetStockParallel(b *testing.B, svc inventoryAPI, productIDs []string) {
	b.Helper()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(runtimeProcIDHint())))
		n := len(productIDs)
		for pb.Next() {
			_ = svc.GetStock(productIDs[r.Intn(n)])
		}
	})
}

func BenchmarkReserveParallel_SafeMutex(b *testing.B) {
	ids, products := benchmarkProducts(4096, 1000)
	svc := NewSafeInventoryService(products)
	benchmarkReserveParallel(b, svc, ids)
}

func BenchmarkReserveParallel_Sharded(b *testing.B) {
	ids, products := benchmarkProducts(4096, 1000)
	svc := NewShardedInventoryService(64, products)
	benchmarkReserveParallel(b, svc, ids)
}

func BenchmarkGetStockParallel_SafeMutex(b *testing.B) {
	ids, products := benchmarkProducts(4096, 1000)
	svc := NewSafeInventoryService(products)
	benchmarkGetStockParallel(b, svc, ids)
}

func BenchmarkGetStockParallel_Sharded(b *testing.B) {
	ids, products := benchmarkProducts(4096, 1000)
	svc := NewShardedInventoryService(64, products)
	benchmarkGetStockParallel(b, svc, ids)
}

func benchmarkProducts(n int, initialStock int) ([]string, map[string]*Product) {
	ids := make([]string, 0, n)
	products := make(map[string]*Product, n)
	for i := 0; i < n; i++ {
		id := "p" + strconv.Itoa(i)
		ids = append(ids, id)
		products[id] = &Product{ID: id, Name: id, Stock: initialStock}
	}
	return ids, products
}

var procHint uint64

// runtimeProcIDHint is a stable-ish per-goroutine seed component without importing unsafe/runtime.
func runtimeProcIDHint() uint64 {
	return atomic.AddUint64(&procHint, 1)
}

