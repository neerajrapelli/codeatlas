package repoingest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
	"unicode"
)

var hostsBySourceType = map[SourceType]map[string]struct{}{
	SourceGitHub: {
		"github.com":     {},
		"www.github.com": {},
	},
	SourceGitLab: {
		"gitlab.com":     {},
		"www.gitlab.com": {},
	},
	SourceBitbucket: {
		"bitbucket.org":     {},
		"www.bitbucket.org": {},
	},
}

var blockedHostSuffixes = []string{
	".local",
	".internal",
	".localhost",
	".lan",
}

// ValidateGitSourceURL blocks SSRF and untrusted git hosts before clone.
func ValidateGitSourceURL(sourceType SourceType, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("sourceUrl is required")
	}
	allowed, ok := hostsBySourceType[sourceType]
	if !ok {
		return fmt.Errorf("unsupported source type for git URL: %s", sourceType)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid sourceUrl: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("sourceUrl must use https (got %q)", u.Scheme)
	}
	if u.User != nil {
		return fmt.Errorf("sourceUrl must not include credentials")
	}
	if u.Fragment != "" || u.RawQuery != "" {
		return fmt.Errorf("sourceUrl must not include query or fragment")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("sourceUrl host is required")
	}
	if _, ok := allowed[host]; !ok {
		return fmt.Errorf("git host %q is not allowed for source type %s", host, sourceType)
	}
	if err := checkHostNotPrivate(host); err != nil {
		return err
	}
	port := u.Port()
	if port != "" && port != "443" {
		return fmt.Errorf("sourceUrl port %q is not allowed", port)
	}
	if err := validateRepoPath(u.Path); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolver := net.Resolver{}
	ips, err := resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("git host resolved to no addresses")
	}
	for _, ia := range ips {
		if isPrivateIP(ia.IP) {
			return fmt.Errorf("git host resolves to private address")
		}
	}
	return nil
}

// ValidateGitBranch rejects branch names that could confuse git CLI.
func ValidateGitBranch(branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil
	}
	if strings.HasPrefix(branch, "-") {
		return fmt.Errorf("invalid branch name")
	}
	for _, r := range branch {
		if r > unicode.MaxASCII {
			return fmt.Errorf("invalid branch name")
		}
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case r == '-', r == '_', r == '/', r == '.':
		default:
			return fmt.Errorf("invalid branch name")
		}
	}
	return nil
}

func validateRepoPath(path string) error {
	path = strings.Trim(path, "/")
	if path == "" {
		return fmt.Errorf("sourceUrl must include owner/repo path")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("sourceUrl path must not contain ..")
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return fmt.Errorf("sourceUrl must include owner/repo path")
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			return fmt.Errorf("invalid path segment in sourceUrl")
		}
	}
	return nil
}

func checkHostNotPrivate(host string) error {
	lower := strings.ToLower(host)
	if lower == "localhost" || lower == "0.0.0.0" {
		return fmt.Errorf("git host not allowed")
	}
	for _, suf := range blockedHostSuffixes {
		if strings.HasSuffix(lower, suf) {
			return fmt.Errorf("git host not allowed")
		}
	}
	if strings.HasPrefix(lower, "metadata.") || lower == "metadata.google.internal" {
		return fmt.Errorf("git host not allowed")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("git host is a private address")
		}
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	ip = ip.To16()
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		privateRanges := []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"127.0.0.0/8",
			"169.254.0.0/16",
			"0.0.0.0/8",
			"100.64.0.0/10", // CGNAT
		}
		for _, cidr := range privateRanges {
			_, network, _ := net.ParseCIDR(cidr)
			if network != nil && network.Contains(ip4) {
				return true
			}
		}
		return false
	}
	// IPv6 ULA, link-local, loopback
	if strings.HasPrefix(ip.String(), "fc") || strings.HasPrefix(ip.String(), "fd") {
		return true
	}
	if strings.HasPrefix(ip.String(), "fe80:") {
		return true
	}
	return false
}
