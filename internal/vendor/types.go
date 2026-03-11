package vendor

// VendorRequest is the payload sent to the vendor check analysis service.
type VendorRequest struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	FrontImage  string  `json:"front_image"`
	BackImage   string  `json:"back_image"`
	TransferID  string  `json:"transfer_id"`
}

// MICRData holds parsed MICR line fields from a check.
type MICRData struct {
	Routing     string `json:"routing"`
	Account     string `json:"account"`
	CheckNumber string `json:"checkNumber"`
}

// VendorResponse is the response from the vendor check analysis service.
type VendorResponse struct {
	// Status: "pass", "fail", "flagged", "reject"
	Status        string   `json:"status"`
	Reason        string   `json:"reason,omitempty"`
	Message       string   `json:"message,omitempty"`
	IQScore       float64  `json:"iqScore,omitempty"`
	MICR          *MICRData `json:"micr,omitempty"`
	Amount        float64  `json:"amount,omitempty"`
	OCRAmount     float64  `json:"ocrAmount,omitempty"`
	EnteredAmount float64  `json:"enteredAmount,omitempty"`
	TransactionID string   `json:"transactionId,omitempty"`
}
