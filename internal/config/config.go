package config

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type stringSlice []*url.URL

func (ss *stringSlice) String() string {
	if ss == nil || len(*ss) == 0 {
		return ""
	}

	urls := make([]string, 0, len(*ss))
	for _, u := range *ss {
		if u != nil {
			urls = append(urls, u.String())
		}
	}

	return strings.Join(urls, ",")
}

func (ss *stringSlice) Set(value string) error {
	*ss = nil

	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)

		if trimmed == "" {
			return fmt.Errorf("URL list contains an empty item (check for trailing or duplicate commas)")
		}

		parsedURL, err := url.ParseRequestURI(trimmed)
		if err != nil {
			return fmt.Errorf("failed to parse URL %s: %w", trimmed, err)
		}

		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("URL %q must start with http:// or https://", trimmed)
		}

		*ss = append(*ss, parsedURL)
	}

	return nil
}

type Config struct {
	URLs       stringSlice
	ReqDepth   int
	Timeout    time.Duration
	ReqTimeout time.Duration
	OutputFile string
	LogFile    string
}

func Load() *Config {
	conf := Config{}

	fs := flag.NewFlagSet("crawler", flag.ExitOnError)

	fs.Var(
		&conf.URLs, "urls", "Comma-separated list of URLs to parse",
	)
	fs.IntVar(
		&conf.ReqDepth, "depth", 1, "The depth of recursive search in any page",
	)
	fs.DurationVar(
		&conf.Timeout, "timeout", time.Minute, "Max time for all program work",
	)
	fs.DurationVar(
		&conf.ReqTimeout, "request-timeout", time.Second*5,
		"Max time for handling one request",
	)
	fs.StringVar(
		&conf.OutputFile, "output", "result.json",
		"File where the result json tree is to be placed",
	)
	fs.StringVar(
		&conf.LogFile, "log", "result.log",
		"File where program work logs are to be placed",
	)

	_ = fs.Parse(os.Args[1:])

	return &conf
}
