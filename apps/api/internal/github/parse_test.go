package github

import "testing"

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		raw       string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{"https://github.com/owner/repo", "owner", "repo", false},
		{"https://github.com/owner/repo.git", "owner", "repo", false},
		{"github.com/acme/app", "acme", "app", false},
		{"", "", "", true},
		{"https://gitlab.com/o/r", "", "", true},
		{"https://github.com/only", "", "", true},
	}
	for _, tc := range tests {
		owner, name, err := ParseRepoURL(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseRepoURL(%q) expected error", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseRepoURL(%q): %v", tc.raw, err)
		}
		if owner != tc.wantOwner || name != tc.wantName {
			t.Fatalf("ParseRepoURL(%q) = %q,%q want %q,%q", tc.raw, owner, name, tc.wantOwner, tc.wantName)
		}
	}
}
