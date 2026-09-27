package models

import (
	"io"
	"net/url"
)

type Node struct {
	Resource string  `json:"resource"`
	Title    string  `json:"title"`
	Links    []*Node `json:"links"`
}

type ParsedPage struct {
	Title string
	Links []*url.URL
}

type FetchResult struct {
	StatusCode  int
	ContentType string
	Body        io.ReadCloser
}
