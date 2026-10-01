package crawler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
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

	baseHost := strings.ToLower(baseURL.Hostname())
	targetHost := strings.ToLower(resolvedTarget.Hostname())

	if baseHost == targetHost {
		return true
	}

	return strings.HasSuffix(targetHost, "."+baseHost)
}

func (c *Crawler) Run(ctx context.Context) []*models.Node {
	var wg sync.WaitGroup
	safeRoots := make([]*SafeNode, len(c.urls))

	for i, startURL := range c.urls {
		urlStr := startURL.String()
		node := NewSafeNode(urlStr)

		safeRoots[i] = node

		if !c.markVisited(urlStr) {
			wg.Add(1)
			go c.crawl(ctx, startURL, node, 0, &wg)
		}
	}

	wg.Wait()

	roots := make([]*models.Node, len(safeRoots))
	for i, safeRoot := range safeRoots {
		roots[i] = safeRoot.RawNode()
	}
	return roots
}

func (c *Crawler) crawl(ctx context.Context, currentURL *url.URL, node *SafeNode, depth int, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := ctx.Err(); err != nil {
		return
	}

	select {
	case <-ctx.Done():
		return
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	}

	urlStr := currentURL.String()

	reqCtx, cancel := context.WithTimeout(ctx, c.reqTimeout)
	defer cancel()

	fetchResult, err := c.fetcher.Fetch(reqCtx, urlStr)
	if err != nil {
		c.logger.Error("fetching failure", slog.String("url", urlStr), slog.Any("error", err))
		return
	}
	defer fetchResult.Body.Close()

	if fetchResult.StatusCode != http.StatusOK {
		c.logger.Debug("status code is not 200", slog.String("url", urlStr), slog.Int("status-code", fetchResult.StatusCode))
		return
	}

	if !strings.HasPrefix(fetchResult.ContentType, "text/html") {
		c.logger.Debug("content-type is not text/html", slog.String("url", urlStr), slog.String("content-type", fetchResult.ContentType))
		return
	}

	parsedPage, err := c.parser.Parse(fetchResult.Body, currentURL)
	if err != nil {
		c.logger.Error("parsing failure", slog.String("url", urlStr), slog.Any("error", err))
		return
	}

	node.SetTitle(parsedPage.Title)

	if depth >= c.maxDepth {
		return
	}

	for _, linkURL := range parsedPage.Links {
		if !c.isSameDomain(currentURL, linkURL) {
			continue
		}

		resolvedLink := currentURL.ResolveReference(linkURL)
		linkStr := resolvedLink.String()

		if c.markVisited(linkStr) {
			continue
		}

		childNode := node.AddChild(linkStr)

		wg.Add(1)
		go c.crawl(ctx, resolvedLink, childNode, depth+1, wg)
	}
}
