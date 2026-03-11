package transfer

import (
	"fmt"
	"time"
)

// Transfer represents a check deposit transfer.
type Transfer struct {
	ID               string  `json:"id"`
	AccountID        string  `json:"account_id"`
	Amount           float64 `json:"amount"`
	State            State   `json:"state"`
	VendorResponse   string  `json:"vendor_response,omitempty"`
	FrontImagePath   string  `json:"front_image_path,omitempty"`
	BackImagePath    string  `json:"back_image_path,omitempty"`
	MICRData         string  `json:"micr_data,omitempty"`
	OCRAmount        float64 `json:"ocr_amount,omitempty"`
	EnteredAmount    float64 `json:"entered_amount,omitempty"`
	TransactionID    string  `json:"transaction_id,omitempty"`
	ContributionType string  `json:"contribution_type,omitempty"`
	SettlementBatchID string `json:"settlement_batch_id,omitempty"`
	SettlementAckAt  string  `json:"settlement_ack_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

// Transition moves the transfer to the given state, returning an error if invalid.
func (t *Transfer) Transition(dst State) error {
	if err := ValidateTransition(t.State, dst); err != nil {
		return fmt.Errorf("transfer %s: %w", t.ID, err)
	}
	t.State = dst
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}
