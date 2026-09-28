package fetcher

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPFetcher_Fetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotUA := r.Header.Get("User-Agent"); gotUA != "CrawlerBot/1.0" {
			t.Errorf("User-Agent header = %q, want %q", gotUA, "CrawlerBot/1.0")
		}

		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>Hello World</html>"))

		case "/not-found":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))

		case "/redirect":
			w.Header().Set("Location", "/ok")
			w.WriteHeader(http.StatusFound)

		case "/slow":
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	tests := []struct {
		name            string
		path            string
		ctxTimeout      time.Duration
		fetcherTimeout  time.Duration
		wantStatusCode  int
		wantContentType string
		wantBody        string
		wantErr         bool
	}{
		{
			name:            "Successful 200 OK request",
			path:            "/ok",
			ctxTimeout:      2 * time.Second,
			fetcherTimeout:  2 * time.Second,
			wantStatusCode:  http.StatusOK,
			wantContentType: "text/html; charset=utf-8",
			wantBody:        "<html>Hello World</html>",
			wantErr:         false,
		},
		{
			name:            "404 Not Found response",
			path:            "/not-found",
			ctxTimeout:      2 * time.Second,
			fetcherTimeout:  2 * time.Second,
			wantStatusCode:  http.StatusNotFound,
			wantContentType: "text/plain; charset=utf-8",
			wantBody:        "Not Found",
			wantErr:         false,
		},
		{
			name:            "Redirect (302) is not followed automatically",
			path:            "/redirect",
			ctxTimeout:      2 * time.Second,
			fetcherTimeout:  2 * time.Second,
			wantStatusCode:  http.StatusFound,
			wantContentType: "",
			wantBody:        "",
			wantErr:         false,
		},
		{
			name:            "Request timed out via Context",
			path:            "/slow",
			ctxTimeout:      20 * time.Millisecond,
			fetcherTimeout:  2 * time.Second,
			wantStatusCode:  0,
			wantContentType: "",
			wantBody:        "",
			wantErr:         true,
		},
		{
			name:            "Request timed out via Client Timeout",
			path:            "/slow",
			ctxTimeout:      2 * time.Second,
			fetcherTimeout:  20 * time.Millisecond,
			wantStatusCode:  0,
			wantContentType: "",
			wantBody:        "",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewHTTPFetcher(tt.fetcherTimeout)

			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			targetURL := ts.URL + tt.path
			res, err := f.Fetch(ctx, targetURL)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				defer res.Body.Close()

				if res.StatusCode != tt.wantStatusCode {
					t.Errorf("StatusCode = %d, want %d", res.StatusCode, tt.wantStatusCode)
				}

				if res.ContentType != tt.wantContentType {
					t.Errorf("ContentType = %q, want %q", res.ContentType, tt.wantContentType)
				}

				body, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("failed to read response body: %v", err)
				}

				if string(body) != tt.wantBody {
					t.Errorf("Body = %q, want %q", string(body), tt.wantBody)
				}
			}
		})
	}
}

func TestHTTPFetcher_Fetch_InvalidURL(t *testing.T) {
	f := NewHTTPFetcher(1 * time.Second)

	tests := []struct {
		name      string
		targetURL string
	}{
		{
			name:      "Invalid URL scheme",
			targetURL: "cache_object\n",
		},
		{
			name:      "Connection refused URL",
			targetURL: "http://127.0.0.1:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.Fetch(context.Background(), tt.targetURL)
			if err == nil {
				t.Errorf("Fetch() expected error for URL %q, got nil", tt.targetURL)
			}
		})
	}
}
