package crawler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"web-crawler-go-test-proj/internal/crawler"
	"web-crawler-go-test-proj/internal/models"
)

type mockFetcher struct {
	fetchFunc func(ctx context.Context, targetURL string) (*models.FetchResult, error)
}

func (m *mockFetcher) Fetch(ctx context.Context, targetURL string) (*models.FetchResult, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, targetURL)
	}
	return nil, errors.New("not implemented")
}

type mockParser struct {
	parseFunc func(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error)
}

func (m *mockParser) Parse(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
	if m.parseFunc != nil {
		return m.parseFunc(r, baseURL)
	}
	return nil, errors.New("not implemented")
}

func parseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse test URL %s: %v", raw, err)
	}
	return u
}

func createNopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCrawler_Run_BasicCrawl(t *testing.T) {
	startURL := parseURL(t, "https://example.com")
	childURL := parseURL(t, "https://example.com/page1")

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, targetURL string) (*models.FetchResult, error) {
			return &models.FetchResult{
				StatusCode:  http.StatusOK,
				ContentType: "text/html; charset=utf-8",
				Body:        io.NopCloser(strings.NewReader("<html></html>")),
			}, nil
		},
	}

	parser := &mockParser{
		parseFunc: func(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
			if baseURL.String() == startURL.String() {
				return &models.ParsedPage{
					Title: "Root Page",
					Links: []*url.URL{childURL},
				}, nil
			}
			return &models.ParsedPage{
				Title: "Child Page",
				Links: []*url.URL{},
			}, nil
		},
	}

	c := crawler.NewCrawler(createNopLogger(), parser, fetcher, []*url.URL{startURL}, 2, time.Second)
	nodes := c.Run(context.Background())

	if len(nodes) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(nodes))
	}

	rootNode := nodes[0]
	if rootNode.Title != "Root Page" {
		t.Errorf("expected title 'Root Page', got %q", rootNode.Title)
	}
	if len(rootNode.Links) != 1 {
		t.Fatalf("expected 1 child link, got %d", len(rootNode.Links))
	}
	if rootNode.Links[0].Title != "Child Page" {
		t.Errorf("expected child title 'Child Page', got %q", rootNode.Links[0].Title)
	}
}

func TestCrawler_MaxDepthLimit(t *testing.T) {
	url1 := parseURL(t, "https://example.com/1")
	url2 := parseURL(t, "https://example.com/2")
	url3 := parseURL(t, "https://example.com/3")

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, targetURL string) (*models.FetchResult, error) {
			return &models.FetchResult{
				StatusCode:  http.StatusOK,
				ContentType: "text/html",
				Body:        io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}

	parser := &mockParser{
		parseFunc: func(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
			switch baseURL.String() {
			case url1.String():
				return &models.ParsedPage{Title: "Depth 0", Links: []*url.URL{url2}}, nil
			case url2.String():
				return &models.ParsedPage{Title: "Depth 1", Links: []*url.URL{url3}}, nil
			default:
				return &models.ParsedPage{Title: "Depth 2", Links: []*url.URL{}}, nil
			}
		},
	}

	c := crawler.NewCrawler(createNopLogger(), parser, fetcher, []*url.URL{url1}, 1, time.Second)
	nodes := c.Run(context.Background())

	if len(nodes) == 0 {
		t.Fatal("expected root node, got 0 nodes")
	}

	root := nodes[0]
	if len(root.Links) != 1 {
		t.Fatalf("expected 1 child at depth 0, got %d", len(root.Links))
	}

	depth1Node := root.Links[0]
	if depth1Node.Title != "Depth 1" {
		t.Errorf("expected 'Depth 1' title at depth 1, but got %q", depth1Node.Title)
	}

	if len(depth1Node.Links) != 0 {
		t.Errorf("expected 0 links beyond max depth, got %d", len(depth1Node.Links))
	}
}

func TestCrawler_Deduplication(t *testing.T) {
	urlA := parseURL(t, "https://example.com/a")
	urlB := parseURL(t, "https://example.com/b")

	visitedCount := make(map[string]int)
	var mu sync.Mutex

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, targetURL string) (*models.FetchResult, error) {
			mu.Lock()
			visitedCount[targetURL]++
			mu.Unlock()

			return &models.FetchResult{
				StatusCode:  http.StatusOK,
				ContentType: "text/html",
				Body:        io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}

	parser := &mockParser{
		parseFunc: func(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
			if baseURL.String() == urlA.String() {
				return &models.ParsedPage{Title: "A", Links: []*url.URL{urlB}}, nil
			}
			return &models.ParsedPage{Title: "B", Links: []*url.URL{urlA}}, nil
		},
	}

	c := crawler.NewCrawler(createNopLogger(), parser, fetcher, []*url.URL{urlA}, 5, time.Second)
	_ = c.Run(context.Background())

	mu.Lock()
	defer mu.Unlock()

	if visitedCount[urlA.String()] != 1 {
		t.Errorf("URL A should be fetched exactly once, got %d", visitedCount[urlA.String()])
	}
	if visitedCount[urlB.String()] != 1 {
		t.Errorf("URL B should be fetched exactly once, got %d", visitedCount[urlB.String()])
	}
}

func TestCrawler_IgnoresExternalDomains(t *testing.T) {
	internalURL := parseURL(t, "https://example.com/internal")
	externalURL := parseURL(t, "https://otherdomain.com/page")

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, targetURL string) (*models.FetchResult, error) {
			return &models.FetchResult{
				StatusCode:  http.StatusOK,
				ContentType: "text/html",
				Body:        io.NopCloser(strings.NewReader("")),
			}, nil
		},
	}

	parser := &mockParser{
		parseFunc: func(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
			return &models.ParsedPage{
				Title: "Main",
				Links: []*url.URL{externalURL},
			}, nil
		},
	}

	c := crawler.NewCrawler(createNopLogger(), parser, fetcher, []*url.URL{internalURL}, 3, time.Second)
	nodes := c.Run(context.Background())

	root := nodes[0]
	if len(root.Links) != 0 {
		t.Errorf("external link should be filtered out, but found %d links", len(root.Links))
	}
}

func TestCrawler_HandlesNon200AndWrongContentType(t *testing.T) {
	startURL := parseURL(t, "https://example.com")

	tests := []struct {
		name        string
		statusCode  int
		contentType string
	}{
		{
			name:        "Non 200 Status Code",
			statusCode:  http.StatusNotFound,
			contentType: "text/html",
		},
		{
			name:        "Non HTML Content Type",
			statusCode:  http.StatusOK,
			contentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := &mockFetcher{
				fetchFunc: func(ctx context.Context, targetURL string) (*models.FetchResult, error) {
					return &models.FetchResult{
						StatusCode:  tt.statusCode,
						ContentType: tt.contentType,
						Body:        io.NopCloser(strings.NewReader("")),
					}, nil
				},
			}

			parser := &mockParser{
				parseFunc: func(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
					t.Fatal("parser should not be called when fetch results are invalid")
					return nil, nil
				},
			}

			c := crawler.NewCrawler(createNopLogger(), parser, fetcher, []*url.URL{startURL}, 2, time.Second)
			nodes := c.Run(context.Background())

			if len(nodes) == 0 {
				t.Fatal("expected node in results map")
			}
			if nodes[0].Title != "" {
				t.Errorf("expected empty title when page fails checks, got %q", nodes[0].Title)
			}
		})
	}
}

func TestCrawler_ContextCancellation(t *testing.T) {
	startURL := parseURL(t, "https://example.com")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, targetURL string) (*models.FetchResult, error) {
			t.Fatal("fetcher should not be called when context is cancelled")
			return nil, nil
		},
	}

	c := crawler.NewCrawler(createNopLogger(), &mockParser{}, fetcher, []*url.URL{startURL}, 2, time.Second)
	_ = c.Run(ctx)
}
