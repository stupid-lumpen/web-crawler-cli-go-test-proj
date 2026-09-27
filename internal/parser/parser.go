package parser

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"web-crawler-go-test-proj/internal/models"

	"github.com/PuerkitoBio/goquery"
)

type HTMLParser struct{}

func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

func (p *HTMLParser) Parse(r io.Reader, baseURL *url.URL) (*models.ParsedPage, error) {
	if baseURL == nil {
		return nil, fmt.Errorf("baseURL cannot be nil")
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse html page from %s: %w", baseURL, err)
	}

	page := &models.ParsedPage{}

	titleSel := doc.Find("title")
	if titleSel.Length() == 0 {
		page.Title = "Unnamed"
	} else {
		page.Title = strings.TrimSpace(titleSel.Text())
	}

	var links []*url.URL

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)

		if href == "" ||
			strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "tel:") {

			return
		}

		ref, err := url.Parse(href)
		if err != nil {
			return
		}

		ref = baseURL.ResolveReference(ref)
		ref.Fragment = ""

		links = append(links, ref)
	})

	page.Links = links

	return page, nil
}
