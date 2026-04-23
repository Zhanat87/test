# REVIEW.md

## Race Condition 1: `InventoryService.GetStock` reads `product.Stock` without synchronization
- Code:
  - `product := s.products[productID]`
  - `return product.Stock`
- What happens:
  - `GetStock` can read `Stock` concurrently with `Reserve` / `ReserveMultiple` writing `Stock`.
  - This is a data race on `product.Stock` and can return stale/incorrect values.
- Production scenario:
  - Goroutine A: loops calling `GetStock("p1")` to show stock in an API response.
  - Goroutine B: concurrently calls `Reserve("p1", 1)` for incoming orders.
  - A reads `Stock` while B writes it → race + inconsistent stock in responses.
- Fix approach:
  - Protect all reads of `Stock` with a shared lock (`RLock`) and all writes with an exclusive lock (`Lock`) using a mutex stored on the service (shared by all calls).

## Race Condition 2: `InventoryService.Reserve` check-then-update is not atomic across goroutines
- Code:
  - `if product.Stock < quantity { ... }`
  - `product.Stock -= quantity`
- What happens:
  - Two goroutines can both pass the check based on the same `Stock` value, then both decrement.
  - This causes overselling / negative stock and is also a data race (concurrent read/write of `Stock`).
- Production scenario:
  - Product stock is 1.
  - Goroutine A and B both call `Reserve("p1", 1)` at the same time.
  - Both see `Stock == 1` and succeed; final stock becomes -1 (oversold by 1).
- Fix approach:
  - Put the check and decrement in the same critical section under a single mutex (`Lock` around both operations).

## Race Condition 3: `InventoryService.ReserveMultiple` has a race and non-atomic “check then reserve”
- Code:
  - First loop checks: `if product.Stock < item.Quantity { return ... }`
  - Second loop updates: `Stock -= item.Quantity`
- What happens:
  - Between the “check all” loop and the “reserve all” loop, other goroutines can change stock.
  - Result: the function may pass checks, then reserve into negative stock, or partially apply changes relative to the real-time state.
  - Also data races on `Stock` (concurrent reads/writes).
- Production scenario:
  - Stock(A)=10, Stock(B)=5.
  - Goroutine A starts `ReserveMultiple([{A:8},{B:5}])` and passes checks.
  - Before it reaches the second loop, Goroutine B reserves B:1.
  - Now A’s second loop decrements B by 5 anyway → B goes negative (oversell), and the operation is not “all-or-nothing” vs the actual concurrent state.
- Fix approach:
  - Use one exclusive lock for the whole operation: validate all items and apply all decrements under the same `Lock`.

## Race Condition 4: `InventoryService.SafeReserve` uses a brand-new mutex per call (no shared mutual exclusion)
- Code:
  - `var mu sync.Mutex` declared inside `SafeReserve`
  - `mu.Lock()` / `mu.Unlock()`
- What happens:
  - Each goroutine calling `SafeReserve` locks a different mutex instance, so goroutines do NOT exclude each other.
  - The code still races on `product.Stock` exactly like `Reserve`.
- Production scenario:
  - 100 goroutines call `SafeReserve("p1", 1)` simultaneously.
  - Every goroutine creates and locks its own `mu`, so they all proceed concurrently and still oversell / data race.
- Fix approach:
  - The mutex must be shared state (field on the service or per-product stored in a map), not a local variable recreated on every call.

