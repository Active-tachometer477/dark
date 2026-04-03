package dark

import (
	"container/list"
	"sync"
)

// lruCache is a concurrency-safe LRU cache with O(1) get/put/evict.
// A maxSize of 0 disables caching (all operations are no-ops).
type lruCache[V any] struct {
	mu      sync.Mutex
	maxSize int
	items   map[string]*list.Element
	order   *list.List // front = most recently used
}

type lruEntry[V any] struct {
	key   string
	value V
}

func newLRUCache[V any](maxSize int) *lruCache[V] {
	return &lruCache[V]{
		maxSize: maxSize,
		items:   make(map[string]*list.Element),
		order:   list.New(),
	}
}

// get returns the cached value and true, or the zero value and false.
// Promotes the entry to the front (most recently used).
func (c *lruCache[V]) get(key string) (V, bool) {
	if c.maxSize <= 0 {
		var zero V
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*lruEntry[V]).value, true
	}
	var zero V
	return zero, false
}

// put adds or updates an entry. Evicts the least recently used entry if full.
func (c *lruCache[V]) put(key string, value V) {
	if c.maxSize <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*lruEntry[V]).value = value
		return
	}
	if c.order.Len() >= c.maxSize {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*lruEntry[V]).key)
		}
	}
	el := c.order.PushFront(&lruEntry[V]{key: key, value: value})
	c.items[key] = el
}

// clear removes all entries.
func (c *lruCache[V]) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element)
	c.order.Init()
}
