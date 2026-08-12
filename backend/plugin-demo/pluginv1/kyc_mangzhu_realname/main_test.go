package main

import "testing"

// TestValidCallbackURL covers M12: only absolute http(s) URLs with a host may
// be forwarded to the KYC provider as a callback/redirect target.
func TestValidCallbackURL(t *testing.T) {
	valid := []string{
		"https://host.example/kyc/cb",
		"http://host.example/kyc/cb?x=1",
	}
	invalid := []string{
		"",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"//host.example/path",
		"/relative/path",
		"https://",
		"ftp://host.example/path",
	}
	for _, u := range valid {
		if !validCallbackURL(u) {
			t.Errorf("validCallbackURL(%q) = false, want true", u)
		}
	}
	for _, u := range invalid {
		if validCallbackURL(u) {
			t.Errorf("validCallbackURL(%q) = true, want false", u)
		}
	}
}

// TestParseThreeFactorResult guards against the KYC bypass (H1): the
// three-factor parser must not treat a transport-level code==200 as VERIFIED
// when the upstream reports a business failure (data.result != 1), matching
// the two-factor parser's behavior.
func TestParseThreeFactorResult(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantOK  bool
	}{
		{
			name:    "business success",
			raw:     `{"code":200,"data":{"result":1,"message":"ok"},"msg":"ok"}`,
			wantOK:  true,
		},
		{
			name:    "http ok but business failed",
			raw:     `{"code":200,"data":{"result":0,"message":"姓名或身份证不匹配"},"msg":"不匹配"}`,
			wantOK:  false,
		},
		{
			name:    "explicit failure code",
			raw:     `{"code":400,"msg":"参数错误"}`,
			wantOK:  false,
		},
		{
			name:    "empty response",
			raw:     ``,
			wantOK:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, msg := parseCodeMsg(c.raw)
			ok, _ := parseThreeFactorResult(code, msg, c.raw)
			if ok != c.wantOK {
				t.Fatalf("parseThreeFactorResult(raw=%q) = %v, want %v", c.raw, ok, c.wantOK)
			}
		})
	}
}

// TestParseThreeFactorResult_MatchesTwoFactor ensures the two parsers agree on
// the same business-failure payload (both must reject result=0).
func TestParseThreeFactorResult_MatchesTwoFactor(t *testing.T) {
	raw := `{"code":200,"data":{"result":0,"message":"不匹配"},"msg":"不匹配"}`
	code, msg := parseCodeMsg(raw)
	threeOK, _ := parseThreeFactorResult(code, msg, raw)
	twoOK, _ := parseTwoFactorResult(code, msg, raw)
	if threeOK != twoOK {
		t.Fatalf("three_factor=%v vs two_factor=%v for identical business-failed payload", threeOK, twoOK)
	}
}
