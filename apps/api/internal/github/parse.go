package github

import (
	"fmt"
	"net/url"
	"strings"
)

// ParseRepoURL extracts owner and repo from a GitHub HTTPS URL.
func ParseRepoURL(raw string) (owner, name string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", fmt.Errorf("empty repository URL")
	}
	if !strings.HasPrefix(raw, "http") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse URL: %w", err)
	}
	host := strings.ToLower(u.Host)
	if host != "github.com" && host != "www.github.com" {
		return "", "", fmt.Errorf("unsupported host %q (GitHub only)", u.Host)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("expected github.com/owner/repo path")
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}
