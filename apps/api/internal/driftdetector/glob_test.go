package driftdetector

import "testing"

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"src/checkout/**", "src/checkout/cart.ts", true},
		{"src/checkout/**", "src/billing/x.ts", false},
		{"**/*.ts", "apps/web/main.ts", true},
		{"src/auth/index.ts", "src/auth/index.ts", true},
	}
	for _, tc := range cases {
		if got := matchGlob(tc.pattern, tc.path); got != tc.want {
			t.Errorf("matchGlob(%q, %q) = %v want %v", tc.pattern, tc.path, got, tc.want)
		}
	}
}
