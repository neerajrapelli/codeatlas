package tenant

import "testing"

func TestNormalize(t *testing.T) {
	if Normalize("") != "default" {
		t.Fatal("empty -> default")
	}
	if Normalize("acme") != "acme" {
		t.Fatal("preserve tenant")
	}
}
