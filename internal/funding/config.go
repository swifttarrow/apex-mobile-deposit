package funding

import (
	"fmt"
	"time"
)

const DepositLimit = 5000.00
const ReturnFee = 30.00

// OmnibusMap maps account IDs to their omnibus account IDs.
var OmnibusMap = map[string]string{
	"ACC-001":        "OMNIBUS-001",
	"ACC-002":        "OMNIBUS-001",
	"ACC-003":        "OMNIBUS-001",
	"ACC-IQA-BLUR":   "OMNIBUS-001",
	"ACC-IQA-GLARE":  "OMNIBUS-001",
	"ACC-MICR-FAIL":  "OMNIBUS-001",
	"ACC-DUP-001":    "OMNIBUS-001",
	"ACC-MISMATCH":   "OMNIBUS-001",
	"ACC-OVER-LIMIT": "OMNIBUS-001",
	"ACC-RETIRE-001": "OMNIBUS-RETIRE",
	"ACC-401K-001":   "OMNIBUS-RETIRE",
}

// AccountTypeMap maps account IDs to their account type (standard, ira, 401k, etc.).
var AccountTypeMap = map[string]string{
	"ACC-RETIRE-001": "retirement",
	"ACC-401K-001":   "401k",
	"ACC-002":        "ira",
}

// AccountDisplayNames maps account IDs to display names for mobile/operator UI.
var AccountDisplayNames = map[string]string{
	"ACC-001":        "Individual Brokerage – ****4821",
	"ACC-002":        "IRA Account – ****3302",
	"ACC-003":        "Joint Account – ****7714",
	"ACC-401K-001":   "401(k) – ****4010",
	"ACC-RETIRE-001": "Retirement – ****9901",
	"ACC-IQA-BLUR":   "Test: IQA Blur",
	"ACC-IQA-GLARE":  "Test: IQA Glare",
	"ACC-MICR-FAIL":  "Test: MICR Fail",
}

// UserAccountIDs maps user IDs to the account IDs that user can access (mobile).
// When nil or when user is not in map, all configured accounts are returned for backward compatibility.
var UserAccountIDs = map[string][]string{
	"alice": {"ACC-001", "ACC-002", "ACC-RETIRE-001", "ACC-MICR-FAIL"}, // includes test account for scenarios
	"bob":   {"ACC-003", "ACC-401K-001"},
}

// TypeRule defines limits for an account type (e.g. 401k contribution limit).
type TypeRule struct {
	// AnnualContributionLimit is the max contribution per year (e.g. 401k $24,500). 0 = use deposit limit only.
	AnnualContributionLimit float64
}

// AccountTypeRules defines per-type rules. Key is account type (401k, ira, standard).
var AccountTypeRules = map[string]TypeRule{
	"401k":       {AnnualContributionLimit: 24500},
	"ira":        {AnnualContributionLimit: 7000},
	"retirement": {AnnualContributionLimit: 7000}, // generic retirement
	"standard":   {}, // uses DepositLimit per deposit
}

// Config holds funding configuration.
type Config struct {
	OmnibusMap          map[string]string
	AccountTypeMap      map[string]string
	AccountDisplayNames map[string]string
	AccountTypeRules    map[string]TypeRule
	UserAccountIDs      map[string][]string // user ID -> account IDs (mobile); nil = no user filtering
	DepositLimit        float64
	ReturnFee           float64
}

// NewConfig creates a new funding configuration with defaults.
func NewConfig() *Config {
	return &Config{
		OmnibusMap:          OmnibusMap,
		AccountTypeMap:      AccountTypeMap,
		AccountDisplayNames: AccountDisplayNames,
		AccountTypeRules:    AccountTypeRules,
		UserAccountIDs:      UserAccountIDs,
		DepositLimit:        DepositLimit,
		ReturnFee:           ReturnFee,
	}
}

// GetOmnibusAccount returns the omnibus account for the given account ID.
// Returns empty string if not found.
func (c *Config) GetOmnibusAccount(accountID string) string {
	return c.OmnibusMap[accountID]
}

// GetAccountIDs returns all configured account IDs (keys of OmnibusMap).
// Used to list deposits across all accounts when account_id is not specified.
func (c *Config) GetAccountIDs() []string {
	ids := make([]string, 0, len(c.OmnibusMap))
	for id := range c.OmnibusMap {
		ids = append(ids, id)
	}
	return ids
}

// GetUserIDForAccount returns the first user (investor) ID that has access to the given account ID.
// Returns empty string if not found or UserAccountIDs is not configured.
func (c *Config) GetUserIDForAccount(accountID string) string {
	if accountID == "" || len(c.UserAccountIDs) == 0 {
		return ""
	}
	for userID, accountIDs := range c.UserAccountIDs {
		for _, id := range accountIDs {
			if id == accountID {
				return userID
			}
		}
	}
	return ""
}

// GetAccountIDsForUser returns account IDs for the given user when UserAccountIDs is configured.
// If userID is empty or UserAccountIDs is nil/empty, returns all configured accounts (backward compatible).
// Only returns IDs that exist in OmnibusMap. Unknown users get an empty slice.
func (c *Config) GetAccountIDsForUser(userID string) []string {
	if userID == "" || len(c.UserAccountIDs) == 0 {
		return c.GetAccountIDs()
	}
	userAccounts, ok := c.UserAccountIDs[userID]
	if !ok {
		return nil
	}
	// Filter to only configured accounts
	out := make([]string, 0, len(userAccounts))
	for _, id := range userAccounts {
		if _, exists := c.OmnibusMap[id]; exists {
			out = append(out, id)
		}
	}
	return out
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

// GetDisplayName returns the display name for an account, or the account ID if not configured.
func (c *Config) GetDisplayName(accountID string) string {
	if name, ok := c.AccountDisplayNames[accountID]; ok {
		return name
	}
	return accountID
}

// GetAnnualContributionLimit returns the annual contribution limit for the account type, or 0 if not applicable.
func (c *Config) GetAnnualContributionLimit(accountID string) float64 {
	t := c.GetAccountType(accountID)
	if rule, ok := c.AccountTypeRules[t]; ok && rule.AnnualContributionLimit > 0 {
		return rule.AnnualContributionLimit
	}
	return 0
}

// CheckLimit returns an error if amount exceeds the deposit limit.
func (c *Config) CheckLimit(amount float64) error {
	if amount > c.DepositLimit {
		return fmt.Errorf("deposit amount exceeds limit: %.2f > %.2f", amount, c.DepositLimit)
	}
	return nil
}

// CurrentYear returns the current calendar year (for YTD contribution checks).
func CurrentYear() int {
	return time.Now().UTC().Year()
}
