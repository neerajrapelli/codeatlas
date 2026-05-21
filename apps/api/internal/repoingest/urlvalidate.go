package repoingest

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var allowedGitHosts = map[string]struct{}{
	"github.com":        {},
	"www.github.com":    {},
	"gitlab.com":        {},
	"www.gitlab.com":    {},
	"bitbucket.org":     {},
	"www.bitbucket.org": {},
}

// ValidateGitSourceURL blocks SSRF and untrusted git hosts before clone.
func ValidateGitSourceURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid sourceUrl: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("sourceUrl must use http or https")
	}
	host := strings.ToLower(u.Hostname())
	if _, ok := allowedGitHosts[host]; !ok {
		return fmt.Errorf("git host not allowed: %s", host)
	}
	if err := checkHostNotPrivate(host); err != nil {
		return err
	}
	// Resolve hostname to catch DNS rebinding to private IPs at clone time would need extra guard.
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("git host resolves to private address")
		}
	}
	return nil
}

func checkHostNotPrivate(host string) error {
	if host == "localhost" || strings.HasSuffix(host, ".local") {
		return fmt.Errorf("git host not allowed")
	}
	if ip := net.ParseIP(host); ip != nil && isPrivateIP(ip) {
		return fmt.Errorf("git host is a private address")
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"0.0.0.0/8",
	}
	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
