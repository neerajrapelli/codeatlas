package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	apiBase       = "https://api.github.com"
	maxRetries    = 5
	baseBackoffMs = 400
)

// Client talks to GitHub REST API (PAT or app installation token via env).
type Client struct {
	token      string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(token string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) Enabled() bool {
	return strings.TrimSpace(c.token) != ""
}

type CommitListItem struct {
	SHA    string
	Author *User
	Date   time.Time
	Message string
}

type User struct {
	ID    int64
	Login string
	Name  string
	Avatar string
}

type CommitDetail struct {
	SHA       string
	Message   string
	Author    *User
	Date      time.Time
	Files     []CommitFileChange
}

type CommitFileChange struct {
	Path      string
	Status    string
	Additions int
	Deletions int
}

type PullRequestItem struct {
	Number    int
	Title     string
	State     string
	User      *User
	CreatedAt time.Time
	MergedAt  *time.Time
	ClosedAt  *time.Time
	Additions int
	Deletions int
	ChangedFiles int
}

type PRFileChange struct {
	Path      string
	Status    string
	Additions int
	Deletions int
}

type IssueItem struct {
	Number    int
	Title     string
	Body      string
	State     string
	CreatedAt time.Time
	User      *User
}

type IssueCommentItem struct {
	ID        int64
	Body      string
	CreatedAt time.Time
	User      *User
}

// ListCommits pages through commit history (newest first).
func (c *Client) ListCommits(ctx context.Context, owner, repo string, since time.Time, maxPages int) ([]CommitListItem, error) {
	var all []CommitListItem
	page := 1
	perPage := 100
	for page <= maxPages {
		q := fmt.Sprintf("%s/repos/%s/%s/commits?per_page=%d&page=%d", apiBase, owner, repo, perPage, page)
		if !since.IsZero() {
			q += "&since=" + since.UTC().Format(time.RFC3339)
		}
		body, err := c.get(ctx, q)
		if err != nil {
			return nil, err
		}
		var pageItems []struct {
			SHA    string `json:"sha"`
			Commit struct {
				Message string `json:"message"`
				Author  struct {
					Date  string `json:"date"`
					Name  string `json:"name"`
					Email string `json:"email"`
				} `json:"author"`
			} `json:"commit"`
			Author *struct {
				ID        int64  `json:"id"`
				Login     string `json:"login"`
				AvatarURL string `json:"avatar_url"`
			} `json:"author"`
		}
		if err := json.Unmarshal(body, &pageItems); err != nil {
			return nil, fmt.Errorf("decode commits: %w", err)
		}
		if len(pageItems) == 0 {
			break
		}
		for _, item := range pageItems {
			t, _ := time.Parse(time.RFC3339, item.Commit.Author.Date)
			var u *User
			if item.Author != nil {
				u = &User{ID: item.Author.ID, Login: item.Author.Login, Avatar: item.Author.AvatarURL}
			}
			msg := item.Commit.Message
			if len(msg) > 240 {
				msg = msg[:240]
			}
			all = append(all, CommitListItem{
				SHA: item.SHA,
				Author: u,
				Date: t,
				Message: msg,
			})
		}
		if len(pageItems) < perPage {
			break
		}
		page++
	}
	return all, nil
}

// GetCommit fetches per-file stats for a single commit.
func (c *Client) GetCommit(ctx context.Context, owner, repo, sha string) (*CommitDetail, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/commits/%s", apiBase, owner, repo, sha)
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var raw struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		Author *struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
		} `json:"author"`
		Files []struct {
			Filename  string `json:"filename"`
			Status    string `json:"status"`
			Additions int    `json:"additions"`
			Deletions int    `json:"deletions"`
		} `json:"files"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339, raw.Commit.Author.Date)
	out := &CommitDetail{SHA: raw.SHA, Message: raw.Commit.Message, Date: t}
	if raw.Author != nil {
		out.Author = &User{ID: raw.Author.ID, Login: raw.Author.Login}
	}
	for _, f := range raw.Files {
		out.Files = append(out.Files, CommitFileChange{
			Path: f.Filename, Status: f.Status,
			Additions: f.Additions, Deletions: f.Deletions,
		})
	}
	return out, nil
}

// ListPullRequests returns closed+open PRs (state=all).
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, maxPages int) ([]PullRequestItem, error) {
	var all []PullRequestItem
	page := 1
	perPage := 100
	for page <= maxPages {
		url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=all&per_page=%d&page=%d&sort=updated&direction=desc",
			apiBase, owner, repo, perPage, page)
		body, err := c.get(ctx, url)
		if err != nil {
			return nil, err
		}
		var pageItems []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			State  string `json:"state"`
			User   *struct {
				ID    int64  `json:"id"`
				Login string `json:"login"`
			} `json:"user"`
			CreatedAt string `json:"created_at"`
			MergedAt  *string `json:"merged_at"`
			ClosedAt  *string `json:"closed_at"`
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
			ChangedFiles int `json:"changed_files"`
		}
		if err := json.Unmarshal(body, &pageItems); err != nil {
			return nil, err
		}
		if len(pageItems) == 0 {
			break
		}
		for _, pr := range pageItems {
			created, _ := time.Parse(time.RFC3339, pr.CreatedAt)
			item := PullRequestItem{
				Number: pr.Number, Title: pr.Title, State: pr.State,
				CreatedAt: created, Additions: pr.Additions,
				Deletions: pr.Deletions, ChangedFiles: pr.ChangedFiles,
			}
			if pr.User != nil {
				item.User = &User{ID: pr.User.ID, Login: pr.User.Login}
			}
			if pr.MergedAt != nil && *pr.MergedAt != "" {
				t, _ := time.Parse(time.RFC3339, *pr.MergedAt)
				item.MergedAt = &t
			}
			if pr.ClosedAt != nil && *pr.ClosedAt != "" {
				t, _ := time.Parse(time.RFC3339, *pr.ClosedAt)
				item.ClosedAt = &t
			}
			all = append(all, item)
		}
		if len(pageItems) < perPage {
			break
		}
		page++
	}
	return all, nil
}

// ListPRFiles returns files changed in a PR.
func (c *Client) ListPRFiles(ctx context.Context, owner, repo string, prNumber int) ([]PRFileChange, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=100", apiBase, owner, repo, prNumber)
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := make([]PRFileChange, 0, len(raw))
	for _, f := range raw {
		out = append(out, PRFileChange{
			Path: f.Filename, Status: f.Status,
			Additions: f.Additions, Deletions: f.Deletions,
		})
	}
	return out, nil
}

// ListIssues returns repository issues (excludes pull requests).
func (c *Client) ListIssues(ctx context.Context, owner, repo string, maxPages int) ([]IssueItem, error) {
	var all []IssueItem
	page := 1
	perPage := 100
	for page <= maxPages {
		url := fmt.Sprintf("%s/repos/%s/%s/issues?state=all&per_page=%d&page=%d&sort=updated&direction=desc",
			apiBase, owner, repo, perPage, page)
		body, err := c.get(ctx, url)
		if err != nil {
			return nil, err
		}
		var pageItems []struct {
			Number    int    `json:"number"`
			Title     string `json:"title"`
			Body      string `json:"body"`
			State     string `json:"state"`
			CreatedAt string `json:"created_at"`
			PullRequest *struct {
				URL string `json:"url"`
			} `json:"pull_request"`
			User *struct {
				ID    int64  `json:"id"`
				Login string `json:"login"`
			} `json:"user"`
		}
		if err := json.Unmarshal(body, &pageItems); err != nil {
			return nil, fmt.Errorf("decode issues: %w", err)
		}
		if len(pageItems) == 0 {
			break
		}
		for _, it := range pageItems {
			if it.PullRequest != nil {
				continue
			}
			created, _ := time.Parse(time.RFC3339, it.CreatedAt)
			item := IssueItem{
				Number: it.Number, Title: it.Title, Body: it.Body,
				State: it.State, CreatedAt: created,
			}
			if it.User != nil {
				item.User = &User{ID: it.User.ID, Login: it.User.Login}
			}
			all = append(all, item)
		}
		if len(pageItems) < perPage {
			break
		}
		page++
	}
	return all, nil
}

// ListIssueComments returns conversation comments for an issue or pull request number.
func (c *Client) ListIssueComments(ctx context.Context, owner, repo string, issueNumber int) ([]IssueCommentItem, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=100", apiBase, owner, repo, issueNumber)
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ID        int64  `json:"id"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
		User      *struct {
			ID    int64  `json:"id"`
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode issue comments: %w", err)
	}
	out := make([]IssueCommentItem, 0, len(raw))
	for _, cmt := range raw {
		created, _ := time.Parse(time.RFC3339, cmt.CreatedAt)
		item := IssueCommentItem{ID: cmt.ID, Body: cmt.Body, CreatedAt: created}
		if cmt.User != nil {
			item.User = &User{ID: cmt.User.ID, Login: cmt.User.Login}
		}
		out = append(out, item)
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			sleep := time.Duration(baseBackoffMs*(1<<attempt)) * time.Millisecond
			sleep += time.Duration(rand.Intn(200)) * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(sleep):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == 429 {
			retryAfter := resp.Header.Get("Retry-After")
			if retryAfter != "" {
				if sec, err := strconv.Atoi(retryAfter); err == nil && sec > 0 {
					time.Sleep(time.Duration(sec) * time.Second)
				}
			}
			lastErr = fmt.Errorf("rate limited: status %d", resp.StatusCode)
			c.logger.Warn("github_rate_limit", "status", resp.StatusCode, "url", url, "attempt", attempt+1)
			continue
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("github status %d: %s", resp.StatusCode, truncate(string(data), 200))
		}
		return data, nil
	}
	return nil, fmt.Errorf("github request failed after retries: %w", lastErr)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
