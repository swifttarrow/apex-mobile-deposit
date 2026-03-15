package funding

import (
	"testing"
)

func TestGetAccountIDsForUser(t *testing.T) {
	cfg := NewConfig()

	// No user ID: returns all configured accounts
	all := cfg.GetAccountIDsForUser("")
	if len(all) == 0 {
		t.Fatal("expected all account IDs for empty user")
	}

	// Known user: returns only that user's accounts
	alice := cfg.GetAccountIDsForUser("alice")
	if len(alice) == 0 {
		t.Fatal("alice should have accounts")
	}
	for _, id := range alice {
		if _, ok := cfg.OmnibusMap[id]; !ok {
			t.Errorf("alice account %q not in OmnibusMap", id)
		}
	}

	// Unknown user: returns nil
	unknown := cfg.GetAccountIDsForUser("unknown-user")
	if unknown != nil {
		t.Errorf("unknown user should get nil, got %v", unknown)
	}
}
