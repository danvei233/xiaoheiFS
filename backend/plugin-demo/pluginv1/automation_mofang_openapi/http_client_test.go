package main

import "testing"

// TestShouldRelogin covers L3: ensureToken only re-logins when the upstream
// answers 405 or says "登陆/login". A JWT that expires with a 401 or
// "token expired" message previously left the plugin retrying with a dead
// token forever.
func TestShouldRelogin(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]any
		want bool
	}{
		{"status 405", map[string]any{"status": 405, "msg": "need login"}, true},
		{"status 401", map[string]any{"status": 401, "msg": "Unauthorized"}, true},
		{"status float 401", map[string]any{"status": 401.0, "msg": "Unauthorized"}, true},
		{"login msg", map[string]any{"status": 200, "msg": "please login first"}, true},
		{"chinese login msg", map[string]any{"status": 200, "msg": "请先登录"}, true},
		{"token expired", map[string]any{"status": 200, "msg": "token expired"}, true},
		{"jwt invalid", map[string]any{"status": 200, "msg": "invalid jwt token"}, true},
		{"normal error", map[string]any{"status": 200, "msg": "order not found"}, false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldRelogin(c.in); got != c.want {
				t.Fatalf("shouldRelogin(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
