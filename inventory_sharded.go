package inventory

import (
	"hash/fnv"
	"sort"
	"sync"
)

type shard struct {
	mu       sync.RWMutex
	products map[string]*Product
}

// ShardedInventoryService reduces lock contention by splitting products into N shards.
// Each shard has its own RWMutex.
//
// ReserveMultiple is implemented by locking all involved shards in a fixed order
// (by shard index) and holding those locks across the full validate+apply sequence
// to guarantee all-or-nothing semantics without deadlocks.
type ShardedInventoryService struct {
	shards []shard
}

func NewShardedInventoryService(shardCount int, products map[string]*Product) *ShardedInventoryService {
	if shardCount <= 0 {
		shardCount = 32
	}

	s := &ShardedInventoryService{
		shards: make([]shard, shardCount),
	}
	for i := range s.shards {
		s.shards[i].products = make(map[string]*Product)
	}

	for id, p := range products {
		if p == nil {
			continue
		}
		s.shardFor(id).products[id] = p
	}

	return s
}

func (s *ShardedInventoryService) shardIndex(productID string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(productID))
	return int(h.Sum32() % uint32(len(s.shards)))
}

func (s *ShardedInventoryService) shardFor(productID string) *shard {
	return &s.shards[s.shardIndex(productID)]
}

func (s *ShardedInventoryService) GetStock(productID string) int {
	sh := s.shardFor(productID)
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	p := sh.products[productID]
	if p == nil {
		return 0
	}
	return p.Stock
}

func (s *ShardedInventoryService) Reserve(productID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	sh := s.shardFor(productID)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	p := sh.products[productID]
	if p == nil {
		return ErrProductNotFound
	}
	if p.Stock < quantity {
		return ErrInsufficientStock
	}
	p.Stock -= quantity
	return nil
}

func (s *ShardedInventoryService) ReserveMultiple(items []ReserveItem) error {
	if len(items) == 0 {
		return nil
	}

	// Determine which shards are involved.
	shardSet := make(map[int]struct{}, len(items))
	for _, item := range items {
		shardSet[s.shardIndex(item.ProductID)] = struct{}{}
	}

	shardIdxs := make([]int, 0, len(shardSet))
	for idx := range shardSet {
		shardIdxs = append(shardIdxs, idx)
	}
	sort.Ints(shardIdxs)

	// Lock all involved shards in canonical order to avoid deadlocks.
	for _, idx := range shardIdxs {
		s.shards[idx].mu.Lock()
	}
	defer func() {
		for i := len(shardIdxs) - 1; i >= 0; i-- {
			s.shards[shardIdxs[i]].mu.Unlock()
		}
	}()

	// Validate + check feasibility while holding all necessary locks.
	for _, item := range items {
		if item.Quantity <= 0 {
			return ErrInvalidQuantity
		}
		sh := &s.shards[s.shardIndex(item.ProductID)]
		p := sh.products[item.ProductID]
		if p == nil {
			return ErrProductNotFound
		}
		if p.Stock < item.Quantity {
			return ErrInsufficientStock
		}
	}

	// Apply all reservations (still holding locks) => all-or-nothing.
	for _, item := range items {
		sh := &s.shards[s.shardIndex(item.ProductID)]
		sh.products[item.ProductID].Stock -= item.Quantity
	}

	return nil
}

