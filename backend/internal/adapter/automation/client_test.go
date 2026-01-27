package automation

import (
	"encoding/json"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	if got := normalizeBaseURL(""); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if got := normalizeBaseURL("https://example.com"); got != "https://example.com/index.php/api/cloud" {
		t.Fatalf("unexpected base url: %q", got)
	}
	if got := normalizeBaseURL("https://example.com/index.php/api/cloud/"); got != "https://example.com/index.php/api/cloud" {
		t.Fatalf("unexpected base url: %q", got)
	}
}

func TestResolveRedirectURL(t *testing.T) {
	base := "https://example.com/index.php/api/cloud"
	if got := resolveRedirectURL(base, "/panel?id=1"); got != "https://example.com/panel?id=1" {
		t.Fatalf("unexpected redirect: %q", got)
	}
	if got := resolveRedirectURL(base, "https://other.test/panel"); got != "https://other.test/panel" {
		t.Fatalf("unexpected redirect: %q", got)
	}
}

func TestParseNetworkStats(t *testing.T) {
	legacy := map[string]any{"BytesSentPersec": 5, "BytesReceivedPersec": 10}
	raw, _ := json.Marshal(legacy)
	in, out := parseNetworkStats(raw)
	if in != 10 || out != 5 {
		t.Fatalf("unexpected legacy stats: %d %d", in, out)
	}

	series := [][]any{{0, 1, 2}, {1, 3, 4}}
	raw, _ = json.Marshal(series)
	in, out = parseNetworkStats(raw)
	if in != 3 || out != 4 {
		t.Fatalf("unexpected series stats: %d %d", in, out)
	}
}
