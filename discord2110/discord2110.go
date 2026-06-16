// Package discord2110 is the library behind the discord2110 command line:
// the HTTP client, rate-limit handling, and the typed data models for the
// Discord REST API v10.
//
// discord2110 is a Discord server archiver. The `invite` command is public
// (no token required). The `crawl` command requires a DISCORD_TOKEN bot token.
//
// Discord is not anti-bot in the Cloudflare sense, but it enforces strict
// per-route rate limits. When a 429 response carries retry_after > MaxWait,
// the client returns errs.RateLimited (exit 5).
package discord2110

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/tamnd/any-cli/kit/errs"
)

const (
	// Host is the Discord API host.
	Host    = "discord.com"
	BaseURL = "https://discord.com/api/v10"
)

// DefaultUserAgent for the bot client.
const DefaultUserAgent = "discord2110/dev (+https://github.com/tamnd/discord2110-cli)"

// Config holds tunable parameters for Client.
type Config struct {
	BaseURL string
	Token   string
	// UserAgent is sent as X-Super-Properties in addition to User-Agent.
	UserAgent string
	Rate      time.Duration
	MaxWait   time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns production-ready defaults.
// It reads DISCORD_TOKEN from the environment.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		Token:     os.Getenv("DISCORD_TOKEN"),
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		MaxWait:   30 * time.Second,
		Retries:   3,
		Timeout:   20 * time.Second,
	}
}

// Client is a rate-limited HTTP client for the Discord API v10.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// request makes an authenticated API request and returns the response body.
// For public endpoints (invite), the token can be empty.
func (c *Client) request(ctx context.Context, path string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, path)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("request %s: %w", path, lastErr)
}

// do performs a single HTTP GET request.
func (c *Client) do(ctx context.Context, path string) (body []byte, retry bool, err error) {
	c.pace()

	url := c.cfg.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")
	if c.cfg.Token != "" {
		req.Header.Set("Authorization", "Bot "+c.cfg.Token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, true, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return b, false, nil
	case http.StatusUnauthorized:
		return nil, false, errs.RateLimited("discord: invalid token (401)")
	case http.StatusForbidden:
		return nil, false, errs.RateLimited("discord: missing permission (403)")
	case http.StatusNotFound:
		return nil, false, errs.NotFound("discord: not found: %s", path)
	case http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(b)
		if retryAfter > 0 && time.Duration(retryAfter)*time.Second <= c.cfg.MaxWait {
			time.Sleep(time.Duration(retryAfter*float64(time.Second)) + 50*time.Millisecond)
			return nil, true, fmt.Errorf("rate limited (retrying after %.1fs)", retryAfter)
		}
		return nil, false, errs.RateLimited("discord: rate limited (retry after %.1fs)", retryAfter)
	default:
		if resp.StatusCode >= 500 {
			return nil, true, fmt.Errorf("http %d", resp.StatusCode)
		}
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
}

// parseRetryAfter parses the retry_after field from a Discord 429 body.
func parseRetryAfter(body []byte) float64 {
	var v struct {
		RetryAfter float64 `json:"retry_after"`
	}
	_ = json.Unmarshal(body, &v)
	return v.RetryAfter
}

// pace blocks until at least Rate has elapsed since the last request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

// backoff returns the wait duration for a retry attempt.
func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
