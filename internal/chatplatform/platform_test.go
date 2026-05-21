package chatplatform

import "testing"

func TestDefaultOpenAPIBaseURLPrioritizesLark(t *testing.T) {
	if got := DefaultOpenAPIBaseURL(""); got != LarkOpenAPIBaseURL {
		t.Fatalf("expected lark default base URL, got %s", got)
	}
	if got := DefaultOpenAPIBaseURL("feishu"); got != FeishuOpenAPIBaseURL {
		t.Fatalf("expected feishu base URL, got %s", got)
	}
}

func TestNormalizePlatformAliases(t *testing.T) {
	cases := map[string]string{
		"":              PlatformLark,
		"larksuite":     PlatformLark,
		"international": PlatformLark,
		"feishu_cn":     PlatformFeishu,
		"china":         PlatformFeishu,
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q)=%q, want %q", input, got, want)
		}
	}
}
