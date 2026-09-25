package crawler

import (
	"io"
	"net/url"

	"web-crawler-go-test-proj/internal/models"
)

type Parser interface {
	Parse(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error)
}
