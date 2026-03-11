package funding

import "fmt"

const DepositLimit = 5000.00
const ReturnFee = 30.00

// OmnibusMap maps account IDs to their omnibus account IDs.
var OmnibusMap = map[string]string{
	"ACC-001":        "OMNIBUS-001",
	"ACC-IQA-BLUR":   "OMNIBUS-001",
	"ACC-IQA-GLARE":  "OMNIBUS-001",
	"ACC-MICR-FAIL":  "OMNIBUS-001",
	"ACC-DUP-001":    "OMNIBUS-001",
	"ACC-MISMATCH":   "OMNIBUS-001",
	"ACC-OVER-LIMIT": "OMNIBUS-001",
	"ACC-RETIRE-001": "OMNIBUS-RETIRE",
}

// AccountTypeMap maps account IDs to their account type.
var AccountTypeMap = map[string]string{
	"ACC-RETIRE-001": "retirement",
}

// Config holds funding configuration.
type Config struct {
	OmnibusMap    map[string]string
	AccountTypeMap map[string]string
	DepositLimit  float64
	ReturnFee     float64
}

// NewConfig creates a new funding configuration with defaults.
func NewConfig() *Config {
	return &Config{
		OmnibusMap:    OmnibusMap,
		AccountTypeMap: AccountTypeMap,
		DepositLimit:  DepositLimit,
		ReturnFee:     ReturnFee,
	}
}

// GetOmnibusAccount returns the omnibus account for the given account ID.
// Returns empty string if not found.
func (c *Config) GetOmnibusAccount(accountID string) string {
	return c.OmnibusMap[accountID]
}

// GetAccountType returns the account type for the given account ID.
func (c *Config) GetAccountType(accountID string) string {
	if t, ok := c.AccountTypeMap[accountID]; ok {
		return t
	}
	return "standard"
}

// GetContributionDefault returns the default contribution type for the account.
// Retirement accounts default to "individual"; others default to "individual".
func (c *Config) GetContributionDefault(accountID string) string {
	_ = c.GetAccountType(accountID) // always individual for now
	return "individual"
}

// CheckLimit returns an error if amount exceeds the deposit limit.
func (c *Config) CheckLimit(amount float64) error {
	if amount > c.DepositLimit {
		return fmt.Errorf("deposit amount exceeds limit: %.2f > %.2f", amount, c.DepositLimit)
	}
	return nil
}
