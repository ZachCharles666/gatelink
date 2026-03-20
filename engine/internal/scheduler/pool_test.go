package scheduler

import (
	"testing"
)

// 测试 AccountInfo 的辅助方法，不依赖 Redis
func TestAccountInfo_ParseFields(t *testing.T) {
	vals := map[string]string{
		fieldVendor:      "anthropic",
		fieldModel:       "claude-sonnet-4-6",
		fieldHealth:      "85.5",
		fieldBalanceUSD:  "100.250000",
		fieldRPMLimit:    "60",
		fieldRPMUsed:     "10",
		fieldDailyLimit:  "50.000000",
		fieldDailyUsed:   "5.000000",
		fieldPeakDaily:   "30.000000",
		fieldStatus:      "active",
		fieldEncryptedKey: "deadbeef",
		fieldSellerID:    "seller-123",
	}

	info, err := parseAccountInfo("acct-001", vals)
	if err != nil {
		t.Fatalf("parseAccountInfo failed: %v", err)
	}

	if info.ID != "acct-001" {
		t.Errorf("ID mismatch: got %s", info.ID)
	}
	if info.Vendor != "anthropic" {
		t.Errorf("Vendor mismatch: got %s", info.Vendor)
	}
	if info.Health != 85.5 {
		t.Errorf("Health mismatch: got %f", info.Health)
	}
	if info.BalanceUSD != 100.25 {
		t.Errorf("BalanceUSD mismatch: got %f", info.BalanceUSD)
	}
	if info.RPMLimit != 60 {
		t.Errorf("RPMLimit mismatch: got %d", info.RPMLimit)
	}
	if info.Status != "active" {
		t.Errorf("Status mismatch: got %s", info.Status)
	}
}

func TestAccountInfo_SafeJSON_NoKey(t *testing.T) {
	info := &AccountInfo{
		ID: "acct-001", Vendor: "openai",
		EncryptedKey: "super-secret-ciphertext",
		Health: 80, BalanceUSD: 50,
	}
	j := info.SafeJSON()
	if len(j) == 0 {
		t.Error("SafeJSON returned empty string")
	}
	// SafeJSON 不得包含 encrypted_key
	if containsSubstring(j, "super-secret") {
		t.Error("SafeJSON must not contain encrypted key content")
	}
}

func TestPoolKey(t *testing.T) {
	k := poolKey("anthropic")
	if k != "pool:anthropic" {
		t.Errorf("unexpected pool key: %s", k)
	}
}

func TestAcctKey(t *testing.T) {
	k := acctKey("uuid-123")
	if k != "acct:uuid-123" {
		t.Errorf("unexpected acct key: %s", k)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}
