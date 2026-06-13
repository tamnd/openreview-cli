// Package openreview is the library behind the orv command: the HTTP client,
// request shaping, and the typed data models for OpenReview.
//
// The public API at api2.openreview.net is open: no authentication key
// required for public conference data. Papers, reviews, and meta-reviews are
// stored as "notes" tagged with invitation strings that identify the venue and
// submission type.
package openreview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to the OpenReview API.
const DefaultUserAgent = "orv/dev (+https://github.com/tamnd/openreview-cli)"

// ErrNotFound is returned when the API returns an empty notes list for an id.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters for NewClient.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api2.openreview.net",
		UserAgent: DefaultUserAgent,
		Rate:      100 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the OpenReview API.
type Client struct {
	base       string
	httpClient *http.Client
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		base:       cfg.BaseURL,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// Search searches OpenReview notes via full-text search.
// Returns papers (up to limit), total count, and any error.
func (c *Client) Search(ctx context.Context, query string, limit, offset int) ([]Paper, int, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("term", query)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	rawURL := c.base + "/notes/search?" + params.Encode()
	var resp apiResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, 0, err
	}

	papers := make([]Paper, len(resp.Notes))
	for i, n := range resp.Notes {
		papers[i] = wireToPaper(n, offset+i+1)
	}
	return papers, resp.Count, nil
}

// Notes fetches notes by invitation string (venue + submission type).
// Returns papers (up to limit), total count, and any error.
func (c *Client) Notes(ctx context.Context, invitation string, limit, offset int) ([]Paper, int, error) {
	if limit <= 0 {
		limit = 20
	}
	params := url.Values{}
	params.Set("invitation", invitation)
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	rawURL := c.base + "/notes?" + params.Encode()
	var resp apiResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, 0, err
	}

	papers := make([]Paper, len(resp.Notes))
	for i, n := range resp.Notes {
		papers[i] = wireToPaper(n, offset+i+1)
	}
	return papers, resp.Count, nil
}

// Note fetches a single note by its OpenReview ID.
// Returns ErrNotFound if the API returns an empty list.
func (c *Client) Note(ctx context.Context, id string) (Paper, error) {
	params := url.Values{}
	params.Set("id", id)

	rawURL := c.base + "/notes?" + params.Encode()
	var resp apiResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return Paper{}, err
	}
	if len(resp.Notes) == 0 {
		return Paper{}, fmt.Errorf("note %q: %w", id, ErrNotFound)
	}
	return wireToPaper(resp.Notes[0], 1), nil
}
