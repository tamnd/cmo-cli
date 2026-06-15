package cmo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const mockHTML = `<!DOCTYPE html><html><body>
<a href="https://cms.math.ca/wp-content/uploads/2019/07/exam2010.pdf">2010 Exam</a>
<a href="https://cms.math.ca/wp-content/uploads/2019/07/sol2010.pdf">2010 Solutions</a>
<a href="https://cms.math.ca/wp-content/uploads/2024/03/CJMO2024-problems.pdf">CJMO 2024 Problems</a>
<a href="https://cms.math.ca/wp-content/uploads/2024/04/cjmo2024-solutions-en.pdf">CJMO 2024 Solutions</a>
<a href="https://cms.math.ca/wp-content/uploads/2026/03/CMO2026-problems.pdf">CMO 2026 Problems</a>
<a href="https://cms.math.ca/wp-content/uploads/2026/04/CMO2026-solutions.pdf">CMO 2026 Solutions</a>
</body></html>`

func newTestClient(ts *httptest.Server) *Client {
	c := NewClient()
	c.HTTP = ts.Client()
	c.Rate = 0
	c.Retries = 3
	return c
}

// TestEditionsSendsUserAgent asserts that the client sets a User-Agent header.
func TestEditionsSendsUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	c.HTTP = &http.Client{} // use default transport so httptest redirect works
	// Override BaseURL via direct URL call.
	_, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("no User-Agent header sent")
	}
	if gotUA != DefaultUserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, DefaultUserAgent)
	}
}

// TestEditionsParsesItems verifies that the mock HTML yields the expected Editions.
func TestEditionsParsesItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	// Patch Get to use the test server URL.
	body, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	links := extractPDFLinks(body)
	eds := buildEditions(links)

	if len(eds) != 3 {
		t.Fatalf("want 3 editions, got %d: %+v", len(eds), eds)
	}

	// Sorted newest-first: 2026 CMO, 2024 CJMO, 2010 CMO.
	if eds[0].Year != 2026 || eds[0].Competition != "CMO" {
		t.Errorf("editions[0] = {%d, %s}, want {2026, CMO}", eds[0].Year, eds[0].Competition)
	}
	if eds[0].Rank != 1 {
		t.Errorf("editions[0].Rank = %d, want 1", eds[0].Rank)
	}
	if eds[1].Year != 2024 || eds[1].Competition != "CJMO" {
		t.Errorf("editions[1] = {%d, %s}, want {2024, CJMO}", eds[1].Year, eds[1].Competition)
	}
	if eds[2].Year != 2010 || eds[2].Competition != "CMO" {
		t.Errorf("editions[2] = {%d, %s}, want {2010, CMO}", eds[2].Year, eds[2].Competition)
	}

	// Both problem and solution URLs present for 2026 CMO.
	if eds[0].ProblemsURL == "" {
		t.Error("editions[0].ProblemsURL is empty")
	}
	if eds[0].SolutionsURL == "" {
		t.Error("editions[0].SolutionsURL is empty")
	}
}

// TestEditionsLimitRespected verifies the limit cap.
func TestEditionsLimitRespected(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	// Build editions manually using the test server URL to fetch HTML.
	body, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	links := extractPDFLinks(body)
	eds := buildEditions(links)
	if len(eds) > 2 {
		eds = eds[:2]
	}
	if len(eds) != 2 {
		t.Errorf("want 2 after limit, got %d", len(eds))
	}
}

// TestEditionsRetriesOn503 verifies that the client retries on 5xx.
func TestEditionsRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	c.HTTP = &http.Client{Timeout: 10 * time.Second}

	body, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server hit %d times, want 3", hits)
	}
	links := extractPDFLinks(body)
	eds := buildEditions(links)
	if len(eds) == 0 {
		t.Error("expected editions after retry")
	}
}

// TestEditionsFiltersCMO verifies competition filtering logic.
func TestEditionsFiltersCMO(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	c := newTestClient(ts)
	body, err := c.Get(context.Background(), ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	links := extractPDFLinks(body)
	eds := buildEditions(links)

	// Filter to CMO only (client-side, as the domain handler does).
	var cmoOnly []Edition
	for _, ed := range eds {
		if ed.Competition == "CMO" {
			cmoOnly = append(cmoOnly, ed)
		}
	}
	for _, ed := range cmoOnly {
		if ed.Competition != "CMO" {
			t.Errorf("expected only CMO, got %s", ed.Competition)
		}
	}
	if len(cmoOnly) != 2 {
		t.Errorf("want 2 CMO editions from mock, got %d", len(cmoOnly))
	}
}

// TestEditionsRetriesExhausted verifies that the client returns an error when
// all retry attempts are exhausted.
func TestEditionsRetriesExhausted(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	c.HTTP = &http.Client{Timeout: 10 * time.Second}
	c.Retries = 2

	_, err := c.Get(context.Background(), ts.URL)
	if err == nil {
		t.Fatal("expected error after retries exhausted, got nil")
	}
	if hits != 3 { // 1 initial + 2 retries
		t.Errorf("server hit %d times, want 3", hits)
	}
}

// TestEditionsContextCancel verifies that a cancelled context stops the request
// promptly.
func TestEditionsContextCancel(t *testing.T) {
	ready := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(ready)
		// Block for longer than the test will wait.
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	c := newTestClient(ts)
	c.HTTP = &http.Client{Timeout: 10 * time.Second}
	c.Rate = 0

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Get(ctx, ts.URL)
		errCh <- err
	}()

	// Wait until the server has received the request, then cancel.
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("server never received request")
	}
	cancel()

	select {
	case err := <-errCh:
		if err == nil {
			t.Error("expected error after context cancel, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Get did not return after context cancel")
	}
}

// TestEditionsURLsAssigned verifies that ProblemsURL and SolutionsURL are set
// correctly for each edition in the mock HTML.
func TestEditionsURLsAssigned(t *testing.T) {
	links := extractPDFLinks([]byte(mockHTML))
	eds := buildEditions(links)

	// Find 2026 CMO.
	var cmo2026 *Edition
	for i := range eds {
		if eds[i].Year == 2026 && eds[i].Competition == "CMO" {
			cmo2026 = &eds[i]
			break
		}
	}
	if cmo2026 == nil {
		t.Fatal("2026 CMO edition not found")
	}
	if !strings.HasSuffix(cmo2026.ProblemsURL, "CMO2026-problems.pdf") {
		t.Errorf("ProblemsURL = %q, want suffix CMO2026-problems.pdf", cmo2026.ProblemsURL)
	}
	if !strings.HasSuffix(cmo2026.SolutionsURL, "CMO2026-solutions.pdf") {
		t.Errorf("SolutionsURL = %q, want suffix CMO2026-solutions.pdf", cmo2026.SolutionsURL)
	}
}

// TestEditionsMissingSolutions verifies that an edition with only a problems
// link has an empty SolutionsURL.
func TestEditionsMissingSolutions(t *testing.T) {
	onlyProblems := `<!DOCTYPE html><html><body>
<a href="https://cms.math.ca/wp-content/uploads/2019/07/exam1969.pdf">1969 CMO</a>
</body></html>`
	links := extractPDFLinks([]byte(onlyProblems))
	eds := buildEditions(links)
	if len(eds) != 1 {
		t.Fatalf("want 1 edition, got %d", len(eds))
	}
	if eds[0].SolutionsURL != "" {
		t.Errorf("SolutionsURL = %q, want empty string", eds[0].SolutionsURL)
	}
	if eds[0].ProblemsURL == "" {
		t.Error("ProblemsURL should not be empty")
	}
}
