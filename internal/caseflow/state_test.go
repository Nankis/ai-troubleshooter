package caseflow

import "testing"

func TestCanTransition(t *testing.T) {
	if !CanTransition(StatusNew, StatusNeedMoreInfo) {
		t.Fatal("NEW should transition to NEED_MORE_INFO")
	}
	if CanTransition(StatusDone, StatusInvestigating) {
		t.Fatal("DONE must be terminal")
	}
}

func TestMissingRequiredFields(t *testing.T) {
	missing := MissingRequiredFields(DomainKline, map[string]string{
		"symbol": "BTCUSDT",
	})
	if len(missing) != 3 {
		t.Fatalf("expected 3 missing fields, got %d: %v", len(missing), missing)
	}

	missing = MissingRequiredFields(DomainAsset, map[string]string{
		"user_id":       "user_1",
		"asset_symbol":  "USDT",
		"abnormal_time": "2026-05-21T20:00:00+08:00",
		"issue_type":    "余额减少",
	})
	if len(missing) != 0 {
		t.Fatalf("expected no missing fields, got %v", missing)
	}
}
