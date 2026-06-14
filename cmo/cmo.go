// Package cmo is the library behind the cmo command line.
// It scrapes the Canadian Mathematical Society's CMO/CJMO archive page,
// extracts all PDF links, and returns typed Edition records.
package cmo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultUserAgent identifies the client to the CMS server.
const DefaultUserAgent = "Mozilla/5.0 (compatible; cmo-cli/dev; +https://github.com/tamnd/cmo-cli)"

// Host is the CMS website this client reads.
const Host = "cms.math.ca"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// archivePath is the single page that lists all CMO/CJMO PDFs.
const archivePath = "/competitions/cmo/"

// Client talks to the CMS website over HTTP.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	last time.Time
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 30 * time.Second},
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
	}
}

// Editions fetches the CMO/CJMO archive page and returns all editions sorted
// newest-first. If limit > 0 the returned slice is capped at limit.
func (c *Client) Editions(ctx context.Context, limit int) ([]Edition, error) {
	body, err := c.Get(ctx, BaseURL+archivePath)
	if err != nil {
		return nil, err
	}
	links := extractPDFLinks(body)
	eds := buildEditions(links)
	if limit > 0 && len(eds) > limit {
		eds = eds[:limit]
	}
	return eds, nil
}

// Get fetches url and returns the response body. It paces and retries according
// to the client's settings.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
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
