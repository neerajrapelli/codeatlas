package vcsauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const clientTimeout = 30 * time.Second

// ListRemoteRepositories fetches repos from the provider API using the user's token.
func ListRemoteRepositories(ctx context.Context, provider Provider, accessToken string, page int) ([]RemoteRepository, error) {
	switch provider {
	case ProviderGitHub:
		return listGitHub(ctx, accessToken, page)
	case ProviderGitLab:
		return listGitLab(ctx, accessToken, page)
	case ProviderBitbucket:
		return listBitbucket(ctx, accessToken, page)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func listGitHub(ctx context.Context, token string, page int) ([]RemoteRepository, error) {
	if page < 1 {
		page = 1
	}
	u := fmt.Sprintf("https://api.github.com/user/repos?per_page=100&page=%d&sort=updated", page)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrNotConnected
	}
	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-Ratelimit-Remaining") == "0" {
		return nil, fmt.Errorf("github rate limit exceeded")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github api %d: %s", resp.StatusCode, string(body))
	}
	var raw []struct {
		ID       int64  `json:"id"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
		HTMLURL  string `json:"html_url"`
		Default  string `json:"default_branch"`
		Private  bool   `json:"private"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]RemoteRepository, 0, len(raw))
	for _, r := range raw {
		out = append(out, RemoteRepository{
			ID:            strconv.FormatInt(r.ID, 10),
			FullName:      r.FullName,
			CloneURL:      r.CloneURL,
			HTMLURL:       r.HTMLURL,
			DefaultBranch: r.Default,
			Private:       r.Private,
		})
	}
	return out, nil
}

func listGitLab(ctx context.Context, token string, page int) ([]RemoteRepository, error) {
	if page < 1 {
		page = 1
	}
	u := fmt.Sprintf("https://gitlab.com/api/v4/projects?membership=true&per_page=100&page=%d&order_by=last_activity_at", page)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("PRIVATE-TOKEN", token)
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrNotConnected
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("gitlab api %d: %s", resp.StatusCode, string(body))
	}
	var raw []struct {
		ID                int64  `json:"id"`
		PathWithNamespace string `json:"path_with_namespace"`
		HTTPURLToRepo     string `json:"http_url_to_repo"`
		WebURL            string `json:"web_url"`
		DefaultBranch     string `json:"default_branch"`
		Visibility        string `json:"visibility"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]RemoteRepository, 0, len(raw))
	for _, r := range raw {
		out = append(out, RemoteRepository{
			ID:            strconv.FormatInt(r.ID, 10),
			FullName:      r.PathWithNamespace,
			CloneURL:      r.HTTPURLToRepo,
			HTMLURL:       r.WebURL,
			DefaultBranch: r.DefaultBranch,
			Private:       r.Visibility != "public",
		})
	}
	return out, nil
}

func listBitbucket(ctx context.Context, token string, page int) ([]RemoteRepository, error) {
	if page < 1 {
		page = 1
	}
	u := fmt.Sprintf("https://api.bitbucket.org/2.0/repositories?role=member&pagelen=100&page=%d", page)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrNotConnected
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("bitbucket api %d: %s", resp.StatusCode, string(body))
	}
	var envelope struct {
		Values []struct {
			UUID     string `json:"uuid"`
			FullName string `json:"full_name"`
			Links    struct {
				Clone []struct {
					Name string `json:"name"`
					Href string `json:"href"`
				} `json:"clone"`
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
			} `json:"links"`
			Mainbranch struct {
				Name string `json:"name"`
			} `json:"mainbranch"`
			IsPrivate bool `json:"is_private"`
		} `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	out := make([]RemoteRepository, 0, len(envelope.Values))
	for _, r := range envelope.Values {
		clone := ""
		for _, c := range r.Links.Clone {
			if c.Name == "https" {
				clone = c.Href
				break
			}
		}
		branch := "main"
		if r.Mainbranch.Name != "" {
			branch = r.Mainbranch.Name
		}
		out = append(out, RemoteRepository{
			ID:            r.UUID,
			FullName:      r.FullName,
			CloneURL:      clone,
			HTMLURL:       r.Links.HTML.Href,
			DefaultBranch: branch,
			Private:       r.IsPrivate,
		})
	}
	return out, nil
}

// ExchangeOAuthCode exchanges authorization code for access token.
func ExchangeOAuthCode(ctx context.Context, provider Provider, cfg OAuthConfig, code string) (accessToken string, refreshToken string, expiresAt *time.Time, scopes []string, err error) {
	switch provider {
	case ProviderGitHub:
		return exchangeGitHub(ctx, cfg, code)
	case ProviderGitLab:
		return exchangeGitLab(ctx, cfg, code)
	case ProviderBitbucket:
		return exchangeBitbucket(ctx, cfg, code)
	default:
		return "", "", nil, nil, fmt.Errorf("unsupported provider")
	}
}

func exchangeGitHub(ctx context.Context, cfg OAuthConfig, code string) (string, string, *time.Time, []string, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURI)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient().Do(req)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", nil, nil, err
	}
	if out.Error != "" || out.AccessToken == "" {
		return "", "", nil, nil, fmt.Errorf("github oauth: %s", out.Error)
	}
	scopes := strings.Fields(strings.ReplaceAll(out.Scope, ",", " "))
	return out.AccessToken, "", nil, scopes, nil
}

func exchangeGitLab(ctx context.Context, cfg OAuthConfig, code string) (string, string, *time.Time, []string, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", cfg.RedirectURI)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://gitlab.com/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient().Do(req)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", nil, nil, err
	}
	if out.Error != "" || out.AccessToken == "" {
		return "", "", nil, nil, fmt.Errorf("gitlab oauth: %s", out.Error)
	}
	var exp *time.Time
	if out.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
		exp = &t
	}
	return out.AccessToken, out.RefreshToken, exp, strings.Fields(out.Scope), nil
}

func exchangeBitbucket(ctx context.Context, cfg OAuthConfig, code string) (string, string, *time.Time, []string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://bitbucket.org/site/oauth2/access_token", strings.NewReader(form.Encode()))
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient().Do(req)
	if err != nil {
		return "", "", nil, nil, err
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scopes       string `json:"scopes"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", nil, nil, err
	}
	if out.Error != "" || out.AccessToken == "" {
		return "", "", nil, nil, fmt.Errorf("bitbucket oauth: %s", out.Error)
	}
	var exp *time.Time
	if out.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
		exp = &t
	}
	return out.AccessToken, out.RefreshToken, exp, strings.Fields(out.Scopes), nil
}

func httpClient() *http.Client {
	return &http.Client{Timeout: clientTimeout}
}

// OAuthConfig holds per-provider OAuth app credentials.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// AuthorizeURL builds provider OAuth authorize URL.
func AuthorizeURL(provider Provider, cfg OAuthConfig, state string) (string, error) {
	switch provider {
	case ProviderGitHub:
		q := url.Values{}
		q.Set("client_id", cfg.ClientID)
		q.Set("redirect_uri", cfg.RedirectURI)
		q.Set("scope", "repo read:user")
		q.Set("state", state)
		return "https://github.com/login/oauth/authorize?" + q.Encode(), nil
	case ProviderGitLab:
		q := url.Values{}
		q.Set("client_id", cfg.ClientID)
		q.Set("redirect_uri", cfg.RedirectURI)
		q.Set("response_type", "code")
		q.Set("scope", "read_api read_repository")
		q.Set("state", state)
		return "https://gitlab.com/oauth/authorize?" + q.Encode(), nil
	case ProviderBitbucket:
		q := url.Values{}
		q.Set("client_id", cfg.ClientID)
		q.Set("response_type", "code")
		q.Set("state", state)
		return "https://bitbucket.org/site/oauth2/authorize?" + q.Encode(), nil
	default:
		return "", fmt.Errorf("unsupported provider")
	}
}
