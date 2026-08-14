package accounts

// UserAccountManager is case (b) of the synthetic-parity corpus: a
// multi-word query like "explore user account" must tokenize this
// CamelCase type name into User/Account/Manager (RESEARCH.md §1-2's two
// tokenizers) and match it, once EXPL-01's multi-word `<query...>` lands.
type UserAccountManager struct {
	Balance int
	Name    string
}

// Describe gives UserAccountManager at least one connected method so the
// type is not a pure, edge-free data type.
func (m *UserAccountManager) Describe() string {
	return m.Name
}
