package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"web-crawler-go-test-proj/internal/models"
)

var (
	ErrStatusCodeNot200   = errors.New("status code must be 200")
	ErrInvalidContentType = errors.New("content-type must be 'text/html'")
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
	}
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

type taskUnit struct {
	url   *url.URL
	depth int
}

func (c *Crawler) fetchAndParse(ctx context.Context, currentURL *url.URL) (*models.ParsedPage, error) {
	urlStr := currentURL.String()
	reqCtx, cancel := context.WithTimeout(ctx, c.reqTimeout)
	defer cancel()

	fetchResult, err := c.fetcher.Fetch(reqCtx, urlStr)
	if err != nil {
		c.logger.Error("fetching failure",
			slog.String("url", urlStr),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to fetch url %q: %w", urlStr, err)
	}
	defer fetchResult.Body.Close()

	if fetchResult.StatusCode != http.StatusOK {
		c.logger.Error("status code is not 200",
			slog.String("url", urlStr),
			slog.Int("status-code", fetchResult.StatusCode))
		return nil, fmt.Errorf(
			"invalid status code %d from %q: %w", fetchResult.StatusCode, urlStr, ErrStatusCodeNot200,
		)
	}

	if !strings.HasPrefix(fetchResult.ContentType, "text/html") {
		c.logger.Error("content-type is not text/html",
			slog.String("url", urlStr),
			slog.String("content-type", fetchResult.ContentType))
		return nil, fmt.Errorf(
			"invalid content-type %s from %q: %w", fetchResult.ContentType, urlStr, ErrInvalidContentType,
		)
	}

	parsedPage, err := c.parser.Parse(fetchResult.Body, currentURL)
	if err != nil {
		c.logger.Error("parsing failure",
			slog.String("url", urlStr),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to parse html from %q: %w", urlStr, err)
	}

	return parsedPage, nil
}

func (c *Crawler) crawl(ctx context.Context, tasks chan *taskUnit, nodesMap *VisitedMap, wg *sync.WaitGroup) {
	for task := range tasks {
		func() {
			defer wg.Done()

			if err := ctx.Err(); err != nil {
				return
			}

			parsedPage, err := c.fetchAndParse(ctx, task.url)
			if err != nil {
				c.logger.Error("handling web-site failure",
					slog.Any("error", err))
				return
			}

			urlStr := task.url.String()

			if err = nodesMap.SetTitle(urlStr, parsedPage.Title); err != nil {
				c.logger.Error("setting title error",
					slog.String("node-url", urlStr),
					slog.Any("error", err))
				return
			}

			if task.depth < c.maxDepth {
				for _, childURL := range parsedPage.Links {
					if !c.isSameDomain(task.url, childURL) {
						continue
					}

					isNew, err := nodesMap.AddChild(urlStr, childURL.String())
					if err != nil {
						continue
					}

					if isNew {
						wg.Add(1)
						select {
						case <-ctx.Done():
							wg.Done()
						case tasks <- &taskUnit{url: childURL, depth: task.depth + 1}:
						}
					}
				}
			}
		}()
	}
}

func (c *Crawler) Run(ctx context.Context) []*models.Node {
	var wg sync.WaitGroup
	safeRoots := NewVisitedMap()
	tasks := make(chan *taskUnit, 100)

	workerCount := 10

	for range workerCount {
		go c.crawl(ctx, tasks, safeRoots, &wg)
	}

	for _, url := range c.urls {
		safeRoots.AddRoot(url.String())
		wg.Add(1)
		select {
		case <-ctx.Done():
			wg.Done()
		case tasks <- &taskUnit{url: url, depth: 0}:
		}
	}

	wg.Wait()
	close(tasks)

	return safeRoots.ToTree()
}
