package accounts

import "errors"

// Validate is one of two identically-named top-level definitions that make
// up case (a) of the behavioral corpus — see orders/validate.go for
// the duplicate. Different package, identical exported symbol name:
// `node Validate` (once NODE-01 lands) must enumerate BOTH as distinct
// definitions rather than resolving to just the lowest-Id one.
func Validate(m *UserAccountManager) error {
	return validateBalance(m.Balance)
}

func validateBalance(balance int) error {
	if balance < 0 {
		return errNegativeBalance
	}
	return nil
}

var errNegativeBalance = errors.New("negative balance")
