package vcsauth

import "testing"

func TestAuthorizeURLGitHub(t *testing.T) {
	url, err := AuthorizeURL(ProviderGitHub, OAuthConfig{
		ClientID:    "cid",
		RedirectURI: "http://localhost:8080/auth/github/callback",
	}, "state123")
	if err != nil {
		t.Fatal(err)
	}
	if url == "" || !contains(url, "github.com") {
		t.Fatalf("unexpected url: %s", url)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
