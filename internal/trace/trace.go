package trace

import (
	"encoding/json"
	"log"
)

// DepositTrace logs a per-deposit decision trace line (JSON, one line).
// Used for observability: inputs → vendor response → business rules → operator actions → settlement status.
// No PII: only synthetic account_id, transfer_id, and status fields.
func DepositTrace(transferID, accountID, stage string, payload map[string]interface{}) {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["event"] = "deposit_trace"
	payload["transfer_id"] = transferID
	payload["account_id"] = accountID
	payload["stage"] = stage
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("deposit_trace marshal: %v", err)
		return
	}
	log.Printf("DEPOSIT_TRACE %s", string(b))
}
