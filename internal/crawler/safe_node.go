package crawler

import (
	"sync"

	"web-crawler-go-test-proj/internal/models"
)

type SafeNode struct {
	mu   sync.RWMutex
	node *models.Node
}

func NewSafeNode(rawURL string) *SafeNode {
	return &SafeNode{
		node: &models.Node{
			Resource: rawURL,
			Links:    make([]*models.Node, 0),
		},
	}
}

func (sn *SafeNode) AddChild(childURL string) *SafeNode {
	childSafe := NewSafeNode(childURL)

	sn.mu.Lock()
	sn.node.Links = append(sn.node.Links, childSafe.node)
	sn.mu.Unlock()

	return childSafe
}

func (sn *SafeNode) SetTitle(title string) {
	sn.mu.Lock()
	defer sn.mu.Unlock()
	sn.node.Title = title
}

func (sn *SafeNode) RawNode() *models.Node {
	sn.mu.RLock()
	defer sn.mu.RUnlock()
	return sn.node
}
