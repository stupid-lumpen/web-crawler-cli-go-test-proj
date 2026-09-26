package models

import "net/url"

type Node struct {
	Resource string  `json:"resource"`
	Title    string  `json:"title"`
	Links    []*Node `json:"links"`
}

type ParsedPage struct {
	Title string
	Links []*url.URL
}
