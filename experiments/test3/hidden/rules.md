# Team rules (hidden from the writers; used only by the grader)

1. Orders must not depend on Billing's internals; components talk only through interfaces (Go) or
   ports (Keel) that the consumer defines.
2. Only adapter/gateway code may open network or database connections; domain code (orders, the
   billing ledger) must not.
3. Billing must never refund more than was charged for an order, and refunds must be greater than 0.
4. A failed audit write must not leave the ledger in a state that disagrees with what the caller
   was told.
