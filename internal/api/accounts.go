package api

import (
	"net/http"

	"github.com/checkstream/checkstream/internal/funding"
	"github.com/checkstream/checkstream/internal/transfer"
)

// AccountsHandler handles GET /accounts for mobile (list accounts with display names and limit info).
type AccountsHandler struct {
	fundingCfg   *funding.Config
	transferRepo *transfer.Repository
}

// NewAccountsHandler creates a new AccountsHandler.
func NewAccountsHandler(fundingCfg *funding.Config, transferRepo *transfer.Repository) *AccountsHandler {
	return &AccountsHandler{fundingCfg: fundingCfg, transferRepo: transferRepo}
}

// AccountSummary is the response shape for one account.
type AccountSummary struct {
	ID                   string  `json:"id"`
	DisplayName          string  `json:"display_name"`
	Type                 string  `json:"type"`
	ContributionLimit    float64 `json:"contribution_limit,omitempty"`    // annual limit if retirement
	YTDContribution      float64 `json:"ytd_contribution,omitempty"`      // sum posted this year
	ContributionRemaining float64 `json:"contribution_remaining,omitempty"` // limit - ytd (if applicable)
}

const userIDHeader = "X-User-ID"

// List handles GET /accounts. Returns accounts for the current user (session or X-User-ID); otherwise all configured accounts.
func (h *AccountsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := ResolveMobileUserID(r)
	accountIDs := h.fundingCfg.GetAccountIDsForUser(userID)
	year := funding.CurrentYear()

	out := make([]AccountSummary, 0, len(accountIDs))
	for _, id := range accountIDs {
		displayName := h.fundingCfg.GetDisplayName(id)
		accType := h.fundingCfg.GetAccountType(id)
		limit := h.fundingCfg.GetAnnualContributionLimit(id)

		summary := AccountSummary{
			ID:          id,
			DisplayName: displayName,
			Type:        accType,
		}
		if limit > 0 {
			summary.ContributionLimit = limit
			ytd, _ := h.transferRepo.SumPostedAmountByAccountYear(id, year)
			summary.YTDContribution = ytd
			if ytd < limit {
				summary.ContributionRemaining = limit - ytd
			}
		}
		out = append(out, summary)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accounts": out,
		"count":    len(out),
	})
}
