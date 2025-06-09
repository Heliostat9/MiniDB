package engine

import (
	"container/list"
)

// Cache implements a simple LRU cache storing SELECT results.
type Cache struct {
	limit int
	size  int
	ll    *list.List
	items map[string]*list.Element
}

type entry struct {
	key   string
	value string
	size  int
}

// NewCache creates a cache with the given size limit in bytes.
func NewCache(limit int) *Cache {
	return &Cache{limit: limit, ll: list.New(), items: make(map[string]*list.Element)}
}

// Get returns a cached value and true if present.
func (c *Cache) Get(k string) (string, bool) {
	if c == nil {
		return "", false
	}
	if e, ok := c.items[k]; ok {
		c.ll.MoveToFront(e)
		return e.Value.(*entry).value, true
	}
	return "", false
}

// Add inserts a key/value pair into the cache.
func (c *Cache) Add(k, v string) {
	if c == nil || c.limit <= 0 {
		return
	}
	if e, ok := c.items[k]; ok {
		c.ll.MoveToFront(e)
		ent := e.Value.(*entry)
		c.size -= ent.size
		ent.value = v
		ent.size = len(v)
		c.size += ent.size
	} else {
		ent := &entry{key: k, value: v, size: len(v)}
		c.items[k] = c.ll.PushFront(ent)
		c.size += ent.size
	}
	for c.size > c.limit {
		c.removeOldest()
	}
}

func (c *Cache) removeOldest() {
	e := c.ll.Back()
	if e == nil {
		return
	}
	c.ll.Remove(e)
	ent := e.Value.(*entry)
	delete(c.items, ent.key)
	c.size -= ent.size
}

var resultCache *Cache

// InitCache initializes the global query result cache.
func InitCache(limit int) {
	resultCache = NewCache(limit)
}
