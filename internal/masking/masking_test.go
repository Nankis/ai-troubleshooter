package masking

import "testing"

func TestMaskString(t *testing.T) {
	got := MaskString("contact user@example.com token: abcdefghijkl 13812345678")
	if got == "contact user@example.com token: abcdefghijkl 13812345678" {
		t.Fatal("expected masking to change sensitive text")
	}
}

func TestMaskValue(t *testing.T) {
	got := MaskValue(map[string]any{
		"email": "user@example.com",
		"note":  "phone 13812345678",
	}).(map[string]any)
	if got["email"] != "[REDACTED]" {
		t.Fatalf("email key should be redacted, got %v", got["email"])
	}
	if got["note"] == "phone 13812345678" {
		t.Fatal("phone in note should be masked")
	}
}

func TestMaskValueHandlesStructsAndTypedSlices(t *testing.T) {
	type payload struct {
		OriginalText string   `json:"original_text"`
		Notes        []string `json:"notes"`
	}
	got := MaskValue(payload{
		OriginalText: "phone 13812345678 token: abcdefghijkl",
		Notes:        []string{"email user@example.com"},
	}).(map[string]any)
	if got["original_text"] == "phone 13812345678 token: abcdefghijkl" {
		t.Fatal("expected struct string field to be masked")
	}
	notes := got["notes"].([]any)
	if notes[0] == "email user@example.com" {
		t.Fatal("expected typed slice element to be masked")
	}
}
