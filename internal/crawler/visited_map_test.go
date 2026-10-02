package crawler_test

import (
	"sync"
	"testing"

	"web-crawler-go-test-proj/internal/crawler"
)

func TestVisitedMap_ConcurrentAccess(t *testing.T) {
	vMap := crawler.NewVisitedMap()
	rootURL := "https://example.com"
	vMap.AddRoot(rootURL)

	var wg sync.WaitGroup
	const numGoroutines = 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(2)

		go func(id int) {
			defer wg.Done()
			_ = vMap.SetTitle(rootURL, "Title")
		}(i)

		go func(id int) {
			defer wg.Done()
			childURL := "https://example.com/child"
			_, _ = vMap.AddChild(rootURL, childURL)
		}(i)
	}

	wg.Wait()

	tree := vMap.ToTree()
	if len(tree) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(tree))
	}

	if tree[0].Title != "Title" {
		t.Errorf("expected title 'Title', got %q", tree[0].Title)
	}
}
