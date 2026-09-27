package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"web-crawler-go-test-proj/internal/models"
)

type HTTPFetcher struct {
	client *http.Client
}

func NewHTTPFetcher(timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (f *HTTPFetcher) Fetch(ctx context.Context, targetURL string) (*models.FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to %s: %w", targetURL, err)
	}

	req.Header.Set("User-Agent", "CrawlerBot/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to %s: %w", targetURL, err)
	}

	return &models.FetchResult{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        resp.Body,
	}, nil
}
