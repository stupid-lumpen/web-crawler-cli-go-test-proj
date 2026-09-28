package crawler

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"sync"
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
	urls       []*url.URL
	maxDepth   int
	reqTimeout time.Duration
	visited    map[string]bool
	mx         sync.Mutex
	sem        chan struct{}
}

func NewCrawler(
	logger *slog.Logger, parser Parser, fetcher Fetcher,
	urls []*url.URL, maxDepth int, reqTimeout time.Duration,
) *Crawler {
	return &Crawler{
		logger:     logger,
		parser:     parser,
		fetcher:    fetcher,
		urls:       urls,
		maxDepth:   maxDepth,
		reqTimeout: reqTimeout,
		visited:    make(map[string]bool),
		sem:        make(chan struct{}, 10),
	}
}

func (c *Crawler) markVisited(rawURL string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()
	if c.visited[rawURL] {
		return true
	}
	c.visited[rawURL] = true
	return false
}

func (c *Crawler) isSameDomain(baseURL, targetURL *url.URL) bool {
	resolvedTarget := baseURL.ResolveReference(targetURL)
	if resolvedTarget.Scheme != "http" && resolvedTarget.Scheme != "https" {
		return false
	}
	return strings.ToLower(baseURL.Hostname()) == strings.ToLower(resolvedTarget.Hostname())
}
