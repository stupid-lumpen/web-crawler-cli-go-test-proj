package parser

import (
	"fmt"
	"io"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"web-crawler-go-test-proj/internal/models"

	"github.com/PuerkitoBio/goquery"
)

func parseRawURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

func TestFindLinks(t *testing.T) {
	baseURL := parseRawURL("https://example.com/blog/posts")

	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name: "Absolute and relative links",
			html: `<html><body>
				<a href="https://example.com/about">About</a>
				<a href="/contact">Contact</a>
				<a href="page2.html">Page 2</a>
			</body></html>`,
			expected: []string{
				"https://example.com/about",
				"https://example.com/contact",
				"https://example.com/blog/page2.html",
			},
		},
		{
			name: "Deduplication and fragment removal",
			html: `<html><body>
				<a href="/about">About 1</a>
				<a href="/about#section1">About 2</a>
				<a href="https://example.com/about">About 3</a>
			</body></html>`,
			expected: []string{
				"https://example.com/about",
			},
		},
		{
			name: "Filter out non-http schemes, empty hrefs and anchors",
			html: `<html><body>
				<a href="">Empty</a>
				<a href="   ">Spaces</a>
				<a href="#anchor">Anchor</a>
				<a href="javascript:void(0)">JS</a>
				<a href="mailto:test@example.com">Mail</a>
				<a href="tel:+123456789">Tel</a>
				<a href="/valid">Valid Link</a>
			</body></html>`,
			expected: []string{
				"https://example.com/valid",
			},
		},
		{
			name:     "No links in HTML",
			html:     `<html><body><p>No links here</p></body></html>`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.html))
			if err != nil {
				t.Fatalf("failed to create goquery document: %v", err)
			}

			links := findLinks(doc, baseURL)

			var got []string
			for _, link := range links {
				got = append(got, link.String())
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("findLinks() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHTMLParser_Parse(t *testing.T) {
	baseURL := parseRawURL("https://example.com")

	tests := []struct {
		name    string
		reader  io.Reader
		base    *url.URL
		want    *models.ParsedPage
		wantErr bool
	}{
		{
			name:   "Valid HTML with title and links",
			reader: strings.NewReader(`<html><head><title>  My Page Title  </title></head><body><a href="/link1">Link 1</a></body></html>`),
			base:   baseURL,
			want: &models.ParsedPage{
				Resource: baseURL,
				Title:    "My Page Title",
				Links:    []*url.URL{parseRawURL("https://example.com/link1")},
			},
			wantErr: false,
		},
		{
			name:   "HTML without title tag defaults to Unnamed",
			reader: strings.NewReader(`<html><body><a href="/link1">Link 1</a></body></html>`),
			base:   baseURL,
			want: &models.ParsedPage{
				Resource: baseURL,
				Title:    "Unnamed",
				Links:    []*url.URL{parseRawURL("https://example.com/link1")},
			},
			wantErr: false,
		},
		{
			name:   "HTML with empty title tag defaults to Unnamed",
			reader: strings.NewReader(`<html><head><title>   </title></head><body></body></html>`),
			base:   baseURL,
			want: &models.ParsedPage{
				Resource: baseURL,
				Title:    "",
				Links:    nil,
			},
			wantErr: false,
		},
		{
			name:    "Nil baseURL returns error",
			reader:  strings.NewReader(`<html><head><title>Title</title></head></html>`),
			base:    nil,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHTMLParser()
			got, err := p.Parse(tt.reader, tt.base)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if got.Title != tt.want.Title {
					t.Errorf("Parse() Title got = %q, want %q", got.Title, tt.want.Title)
				}

				if !reflect.DeepEqual(got.Resource, tt.want.Resource) {
					t.Errorf("Parse() Resource got = %v, want %v", got.Resource, tt.want.Resource)
				}

				var gotLinks, wantLinks []string
				for _, l := range got.Links {
					gotLinks = append(gotLinks, l.String())
				}
				for _, l := range tt.want.Links {
					wantLinks = append(wantLinks, l.String())
				}

				if !reflect.DeepEqual(gotLinks, wantLinks) {
					t.Errorf("Parse() Links got = %v, want %v", gotLinks, wantLinks)
				}
			}
		})
	}
}

// ErrReader эмитирует ошибку чтения потока для проверки обработки ошибок
type errReader struct{}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func TestHTMLParser_Parse_ReaderError(t *testing.T) {
	p := NewHTMLParser()
	baseURL := parseRawURL("https://example.com")

	_, err := p.Parse(&errReader{}, baseURL)
	if err == nil {
		t.Error("Parse() expected error on broken reader, got nil")
	}
}
