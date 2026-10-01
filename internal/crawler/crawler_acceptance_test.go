package crawler_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"web-crawler-go-test-proj/internal/crawler"
	"web-crawler-go-test-proj/internal/fetcher"
	"web-crawler-go-test-proj/internal/models"
	"web-crawler-go-test-proj/internal/parser"
)

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func collectURLs(nodes []*models.Node) []string {
	var urls []string
	var walk func(n *models.Node)

	walk = func(n *models.Node) {
		if n == nil {
			return
		}
		urls = append(urls, n.Resource)
		for _, child := range n.Links {
			walk(child)
		}
	}

	for _, root := range nodes {
		walk(root)
	}
	return urls
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestCrawlerDepthAcceptance(t *testing.T) {
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			fmt.Fprintf(w, `<html><head><title>Root</title></head><body>
				<a href="%s/a">Link A</a>
				<a href="%s/b">Link B</a>
			</body></html>`, ts.URL, ts.URL)
		case "/a":
			fmt.Fprintf(w, `<html><head><title>Page A</title></head><body>
				<a href="%s/c">Link C</a>
			</body></html>`, ts.URL)
		case "/b":
			fmt.Fprintf(w, `<html><head><title>Page B</title></head><body>Page B</body></html>`)
		case "/c":
			fmt.Fprintf(w, `<html><head><title>Page C</title></head><body>Page C</body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	tests := []struct {
		name            string
		depth           int
		expectedVisited int
		expectedURLs    []string
		notExpectedURLs []string
	}{
		{
			name:            "Depth 0 - только стартовая страница",
			depth:           0,
			expectedVisited: 1,
			expectedURLs:    []string{ts.URL + "/"},
			notExpectedURLs: []string{ts.URL + "/a", ts.URL + "/b", ts.URL + "/c"},
		},
		{
			name:            "Depth 1 - стартовая страница и ссылки 1-го уровня",
			depth:           1,
			expectedVisited: 3,
			expectedURLs:    []string{ts.URL + "/", ts.URL + "/a", ts.URL + "/b"},
			notExpectedURLs: []string{ts.URL + "/c"},
		},
		{
			name:            "Depth 2 - корень, 1-й уровень и переход к ссылке /c (2-й уровень)",
			depth:           2,
			expectedVisited: 4,
			expectedURLs:    []string{ts.URL + "/", ts.URL + "/a", ts.URL + "/b", ts.URL + "/c"},
			notExpectedURLs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			reqTimeout := time.Second * 2
			f := fetcher.NewHTTPFetcher(reqTimeout)
			p := parser.NewHTMLParser()
			log := newDiscardLogger() // Используем молчаливый логгер для тестов

			startURL, err := url.Parse(ts.URL + "/")
			if err != nil {
				t.Fatalf("failed to parse start url: %v", err)
			}

			c := crawler.NewCrawler(
				log,
				p,
				f,
				[]*url.URL{startURL},
				tt.depth,
				reqTimeout,
			)

			roots := c.Run(ctx)

			visitedURLs := collectURLs(roots)

			if len(visitedURLs) != tt.expectedVisited {
				t.Errorf("expected %d visited URLs, got %d. Visited: %v", tt.expectedVisited, len(visitedURLs), visitedURLs)
			}

			for _, expectedURL := range tt.expectedURLs {
				if !contains(visitedURLs, expectedURL) {
					t.Errorf("expected URL %s was not visited", expectedURL)
				}
			}

			for _, notExpectedURL := range tt.notExpectedURLs {
				if contains(visitedURLs, notExpectedURL) {
					t.Errorf("URL %s should NOT have been visited at depth %d", notExpectedURL, tt.depth)
				}
			}
		})
	}
}
