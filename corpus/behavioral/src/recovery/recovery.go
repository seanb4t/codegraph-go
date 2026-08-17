package recovery

// recoverAccount is case (c)'s structurally-connected non-test symbol: it
// is called by TestAccountRecovery (recovery_test.go) and itself calls
// validateRecovery — giving the file-relevance gate (EXPL-03) a
// structurally-connected symbol to prefer over the weakly-connected
// Test*-named function in the same cluster.
func recoverAccount(accountID string) bool {
	return validateRecovery(accountID)
}

func validateRecovery(accountID string) bool {
	return accountID != ""
}
