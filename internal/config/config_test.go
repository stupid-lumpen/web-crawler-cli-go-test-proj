package config

import (
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestStringSlice_Set(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "Valid single HTTP URL",
			input:   "http://example.com",
			want:    []string{"http://example.com"},
			wantErr: false,
		},
		{
			name:    "Valid multiple HTTPS URLs with spaces",
			input:   "https://example.com, https://golang.org",
			want:    []string{"https://example.com", "https://golang.org"},
			wantErr: false,
		},
		{
			name:    "Empty input item (leading comma)",
			input:   ",https://example.com",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Empty input item (trailing comma)",
			input:   "https://example.com,",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Empty input item (duplicate comma)",
			input:   "https://example.com,,https://golang.org",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid scheme (ftp)",
			input:   "ftp://example.com",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid URL format",
			input:   "not-a-valid-url",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "URL without scheme",
			input:   "example.com",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ss stringSlice
			err := ss.Set(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				var got []string
				for _, u := range ss {
					got = append(got, u.String())
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Set() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestStringSlice_String(t *testing.T) {
	parseURL := func(raw string) *url.URL {
		u, _ := url.Parse(raw)
		return u
	}

	tests := []struct {
		name  string
		slice stringSlice
		want  string
	}{
		{
			name:  "Nil slice",
			slice: nil,
			want:  "",
		},
		{
			name:  "Empty slice",
			slice: stringSlice{},
			want:  "",
		},
		{
			name:  "Single URL",
			slice: stringSlice{parseURL("http://example.com")},
			want:  "http://example.com",
		},
		{
			name:  "Multiple URLs",
			slice: stringSlice{parseURL("http://example.com"), parseURL("https://golang.org")},
			want:  "http://example.com,https://golang.org",
		},
		{
			name:  "Slice containing nil element",
			slice: stringSlice{parseURL("http://example.com"), nil, parseURL("https://golang.org")},
			want:  "http://example.com,https://golang.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.slice.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	parseURLs := func(urls ...string) stringSlice {
		var res stringSlice
		for _, raw := range urls {
			u, _ := url.Parse(raw)
			res = append(res, u)
		}
		return res
	}

	tests := []struct {
		name string
		args []string
		want *Config
	}{
		{
			name: "Default configuration values",
			args: []string{"cmd"},
			want: &Config{
				URLs:       nil,
				ReqDepth:   1,
				Timeout:    time.Minute,
				ReqTimeout: 5 * time.Second,
				OutputFile: "result.json",
				LogFile:    "result.log",
			},
		},
		{
			name: "Custom flags passed",
			args: []string{
				"cmd",
				"-urls=https://example.com,http://test.com",
				"-depth=3",
				"-timeout=2m",
				"-request-timeout=10s",
				"-output=out.json",
				"-log=app.log",
			},
			want: &Config{
				URLs:       parseURLs("https://example.com", "http://test.com"),
				ReqDepth:   3,
				Timeout:    2 * time.Minute,
				ReqTimeout: 10 * time.Second,
				OutputFile: "out.json",
				LogFile:    "app.log",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			got := Load()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load()\ngot  = %+v\nwant = %+v", got, tt.want)
			}
		})
	}
}
