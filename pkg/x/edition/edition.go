// Package edition defines Kargo distribution editions.
package edition

// Edition identifies the Kargo distribution serving a request.
type Edition string

const (
	// Community is the open-source Kargo distribution.
	Community Edition = "community"
	// Enterprise is the Kargo Enterprise distribution.
	Enterprise Edition = "enterprise"
)

// IsEnterprise reports whether the edition is Enterprise.
func (e Edition) IsEnterprise() bool {
	return e == Enterprise
}
