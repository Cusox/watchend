package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Repository struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"`
	Description   string   `json:"description"`
	HTMLURL       string   `json:"html_url"`
	DefaultBranch string   `json:"default_branch"`
	Stars         int      `json:"stargazers_count"`
	Archived      bool     `json:"archived"`
	Topics        []string `json:"topics"`
	Language      string   `json:"language"`
	License       struct {
		SPDXID string `json:"spdx_id"`
	} `json:"license"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Owner     struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type Star struct {
	StarredAt  time.Time  `json:"starred_at"`
	Repository Repository `json:"repo"`
}

type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

func NewClient(client *http.Client, token string) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{httpClient: client, token: token, baseURL: "https://api.github.com"}
}

func (c *Client) Stars(ctx context.Context) ([]Star, error) {
	var result []Star
	err := c.StarsEach(ctx, func(page []Star, _ int) error { result = append(result, page...); return nil })
	return result, err
}

func (c *Client) StarsEach(ctx context.Context, consume func([]Star, int) error) error {
	path := "/user/starred?sort=created&direction=asc&per_page=100"
	for path != "" {
		var page []Star
		next, total, err := c.get(ctx, path, &page)
		if err != nil {
			return err
		}
		if err = consume(page, total); err != nil {
			return err
		}
		path = next
	}
	return nil
}

func (c *Client) SetBaseURL(value string) { c.baseURL = strings.TrimRight(value, "/") }

func (c *Client) get(ctx context.Context, path string, target any) (string, int, error) {
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		path = c.baseURL + "/" + strings.TrimLeft(path, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Accept", "application/vnd.github.star+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", 0, fmt.Errorf("GitHub API %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err = json.NewDecoder(resp.Body).Decode(target); err != nil {
		return "", 0, err
	}
	link := resp.Header.Get("Link")
	return nextLink(link), linkTotal(link, len(*target.(*[]Star))), nil
}

func nextLink(value string) string {
	for part := range strings.SplitSeq(value, ",") {
		pieces := strings.Split(strings.TrimSpace(part), ";")
		if len(pieces) >= 2 && strings.Contains(pieces[1], `rel="next"`) {
			return strings.Trim(strings.TrimSpace(pieces[0]), "<>")
		}
	}
	return ""
}

func linkTotal(value string, fallback int) int {
	for part := range strings.SplitSeq(value, ",") {
		pieces := strings.Split(strings.TrimSpace(part), ";")
		if len(pieces) < 2 || !strings.Contains(pieces[1], `rel="last"`) {
			continue
		}
		start := strings.Index(pieces[0], "page=")
		if start < 0 {
			continue
		}
		page := strings.Trim(strings.TrimSpace(pieces[0][start+5:]), "<>&")
		if end := strings.IndexAny(page, "&>"); end >= 0 {
			page = page[:end]
		}
		if n, err := strconv.Atoi(page); err == nil {
			return n * 100
		}
	}
	return fallback
}
