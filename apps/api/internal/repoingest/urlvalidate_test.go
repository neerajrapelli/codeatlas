package repoingest

import (
	"strings"
	"testing"
)

func TestValidateGitSourceURL_acceptsGitHubHTTPS(t *testing.T) {
	if err := ValidateGitSourceURL(SourceGitHub, "https://github.com/org/repo"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGitSourceURL_rejectsHTTP(t *testing.T) {
	err := ValidateGitSourceURL(SourceGitHub, "http://github.com/org/repo")
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https required, got %v", err)
	}
}

func TestValidateGitSourceURL_rejectsWrongHostForType(t *testing.T) {
	err := ValidateGitSourceURL(SourceGitHub, "https://gitlab.com/org/repo")
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected host mismatch, got %v", err)
	}
}

func TestValidateGitSourceURL_rejectsCredentials(t *testing.T) {
	err := ValidateGitSourceURL(SourceGitHub, "https://user:pass@github.com/org/repo")
	if err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("expected credentials rejected, got %v", err)
	}
}

func TestValidateGitSourceURL_rejectsPrivateIPLiteral(t *testing.T) {
	err := ValidateGitSourceURL(SourceGitHub, "https://127.0.0.1/org/repo")
	if err == nil {
		t.Fatal("expected private IP rejected")
	}
}

func TestValidateGitSourceURL_rejectsPathTraversal(t *testing.T) {
	err := ValidateGitSourceURL(SourceGitHub, "https://github.com/org/../secret")
	if err == nil {
		t.Fatal("expected .. rejected")
	}
}

func TestValidateGitBranch_rejectsFlagInjection(t *testing.T) {
	if err := ValidateGitBranch("--version"); err == nil {
		t.Fatal("expected branch flag rejected")
	}
}
