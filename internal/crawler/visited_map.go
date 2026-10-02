package crawler

import (
	"fmt"
	"sync"

	"web-crawler-go-test-proj/internal/models"
)

type VisitedMap struct {
	mu       sync.Mutex
	nodes    map[string]*models.Node
	rootURLs map[string]bool
}

func NewVisitedMap() *VisitedMap {
	return &VisitedMap{
		nodes:    make(map[string]*models.Node),
		rootURLs: make(map[string]bool),
	}
}

func (vm *VisitedMap) AddNode(url string) bool {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if _, exists := vm.nodes[url]; exists {
		return false
	}

	vm.nodes[url] = &models.Node{
		Resource: url,
		Links:    make([]*models.Node, 0),
	}
	return true
}

func (vm *VisitedMap) AddRoot(url string) bool {
	isNew := vm.AddNode(url)

	vm.mu.Lock()
	vm.rootURLs[url] = true
	vm.mu.Unlock()

	return isNew
}

func (vm *VisitedMap) AddChild(parentURL, childURL string) (bool, error) {
	isNew := vm.AddNode(childURL)

	vm.mu.Lock()
	defer vm.mu.Unlock()

	parent, pExists := vm.nodes[parentURL]
	if !pExists {
		return false, fmt.Errorf("parent node %q does not exist", parentURL)
	}

	child := vm.nodes[childURL]

	alreadyLinked := false
	for _, link := range parent.Links {
		if link.Resource == childURL {
			alreadyLinked = true
			break
		}
	}

	if !alreadyLinked {
		parent.Links = append(parent.Links, child)
	}

	return isNew, nil
}

func (vm *VisitedMap) SetTitle(url, title string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	node, exists := vm.nodes[url]
	if !exists {
		return fmt.Errorf("node %q does not exist", url)
	}

	node.Title = title
	return nil
}

func (vm *VisitedMap) ToTree() []*models.Node {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if len(vm.nodes) == 0 {
		return nil
	}

	var roots []*models.Node
	for url := range vm.rootURLs {
		if node, exists := vm.nodes[url]; exists {
			roots = append(roots, node)
		}
	}

	return roots
}
