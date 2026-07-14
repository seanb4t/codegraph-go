package ledger

// AccountBalanceHelper is case (d)'s symbol A: its NAME lexically matches
// a query like "account balance", but it is deliberately graph-isolated —
// zero calls in, zero calls out. Once EXPL-02's RWR ranking lands, a
// structurally-connected symbol (ReconcileLedger, below) that does NOT
// lexically match the query must still outrank this one.
func AccountBalanceHelper() string {
	return "account balance helper — deliberately disconnected"
}

// GetBalance is the "true" target of a query like "account balance": it
// does not itself carry heavy connectivity, but delegates into
// ReconcileLedger, the structurally central symbol.
func GetBalance(accountID string) int {
	return ReconcileLedger(accountID)
}

// ReconcileLedger is case (d)'s symbol B: its name does NOT lexically
// match "account balance" at all, but it is heavily structurally
// connected — called by GetBalance and PostTransaction, and itself calls
// applyAdjustment and AuditEntry — giving it strictly more graph edges
// than AccountBalanceHelper (which has zero).
func ReconcileLedger(accountID string) int {
	applyAdjustment(accountID)
	AuditEntry(accountID)
	return 0
}

// PostTransaction is a second caller into ReconcileLedger, reinforcing its
// structural centrality relative to the isolated AccountBalanceHelper.
func PostTransaction(accountID string, amount int) int {
	return ReconcileLedger(accountID)
}

func applyAdjustment(accountID string) {}

// AuditEntry is exported so it also appears as an independently callable
// symbol in the graph, adding to ReconcileLedger's outbound edge count.
func AuditEntry(accountID string) {}
