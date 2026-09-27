package crawler

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"time"

	"web-crawler-go-test-proj/internal/models"
)

type Parser interface {
	Parse(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error)
}

type Fetcher interface {
	Fetch(ctx context.Context, targetURL string) (*models.FetchResult, error)
}

type Crawler struct {
	logger     *slog.Logger
	parser     Parser
	fetcher    Fetcher
	urls       []string
	maxDepth   int
	reqTimeout time.Duration
}

func NewCrawler(
	logger *slog.Logger, parser Parser, fetcher Fetcher,
	urls []string, maxDepth int, reqTimeout time.Duration,
) *Crawler {
	return &Crawler{
		logger:     logger,
		parser:     parser,
		fetcher:    fetcher,
		urls:       urls,
		maxDepth:   maxDepth,
		reqTimeout: reqTimeout,
	}
}
