package inventory

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestReserve_ConcurrentOversell(t *testing.T) {
	svc := NewSafeInventoryService(map[string]*Product{
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

func TestReserveMultiple_Atomicity(t *testing.T) {
	svc := NewSafeInventoryService(map[string]*Product{
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

