# ANSWERS.md

## Q1
Because the mutex is **allocated inside the function**, every call to `SafeReserve` creates a **different** `sync.Mutex` instance. Two goroutines calling `SafeReserve` will each lock their own mutex, so there is **no shared mutual exclusion** protecting `s.products[...]` or `product.Stock`. The lock does not serialize access between goroutines, so the race remains.

## Q2
With per-product locks, Goroutine 1 can lock A then block trying to lock B, while Goroutine 2 locks B then blocks trying to lock A. This is a classic **deadlock** caused by inconsistent lock ordering.

To prevent it:
- Enforce a **global lock ordering** (e.g., always lock products by sorted productID), then acquire locks in that order.
- Or avoid multiple per-product locks by using a single service-level lock for the multi-item operation.

## Q3
It introduces a time-of-check/time-of-use gap:
- The code checks `product.Stock` **without holding the lock**, then later decrements under a lock.
- Another goroutine can reserve in between, so the decrement can happen even though stock is no longer sufficient, leading to **oversell / negative stock**.

It can be worse than no locks because it creates a **false sense of safety** while still allowing the invalid interleaving; you can observe “successful” reservations that violate your invariants even more reliably under load.

## Q4
No. `-race` increases the chance of detecting data races during that particular run, but it is not a proof:
- Race detection depends on **executed interleavings**; some races may not be exercised by the test schedule.
- Some races can be **timing-dependent** and may appear only under different CPU load, different machines, or different inputs.

Absence of warnings means “not detected in this run”, not “race-free”.

