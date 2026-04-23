package inventory

import (
	"errors"
	"sync"
)

type Product struct {
	ID    string
	Name  string
	Stock int
}

type ReserveItem struct {
	ProductID string
	Quantity  int
}

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidQuantity   = errors.New("invalid quantity")
)

type SafeInventoryService struct {
	mu       sync.RWMutex
	products map[string]*Product
}

func NewSafeInventoryService(products map[string]*Product) *SafeInventoryService {
	if products == nil {
		products = make(map[string]*Product)
	}
	return &SafeInventoryService{products: products}
}

func (s *SafeInventoryService) GetStock(productID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product := s.products[productID]
	if product == nil {
		return 0
	}
	return product.Stock
}

func (s *SafeInventoryService) Reserve(productID string, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	product := s.products[productID]
	if product == nil {
		return ErrProductNotFound
	}
	if product.Stock < quantity {
		return ErrInsufficientStock
	}

	product.Stock -= quantity
	return nil
}

func (s *SafeInventoryService) ReserveMultiple(items []ReserveItem) error {
	if len(items) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate + check feasibility first (single critical section => all-or-nothing).
	for _, item := range items {
		if item.Quantity <= 0 {
			return ErrInvalidQuantity
		}
		product := s.products[item.ProductID]
		if product == nil {
			return ErrProductNotFound
		}
		if product.Stock < item.Quantity {
			return ErrInsufficientStock
		}
	}

	for _, item := range items {
		s.products[item.ProductID].Stock -= item.Quantity
	}

	return nil
}

