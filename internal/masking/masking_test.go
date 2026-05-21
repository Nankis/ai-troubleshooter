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
