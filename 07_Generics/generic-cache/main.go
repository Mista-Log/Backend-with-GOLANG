// Project 3: Generic Cache
//
// A key/value cache with TWO type parameters — K for the key, V for the
// value — plus optional per-entry TTL expiry and simple FIFO eviction once
// a capacity limit is reached.
package main

import (
	"fmt"
	"time"
)

// entry[V any] holds a value plus optional expiry metadata. V stays
// unconstrained — the cache never compares or operates ON values, only
// stores and returns them.
type entry[V any] struct {
	value     V
	expiresAt time.Time
	hasExpiry bool
}

// Cache[K comparable, V any] needs K to be `comparable` specifically
// because K is used as a map key internally (recall Module 04: map keys
// must support ==, exactly what `comparable` guarantees) — V has no such
// requirement, so it stays maximally permissive at `any`.
type Cache[K comparable, V any] struct {
	data     map[K]entry[V]
	order    []K // insertion order, oldest first — used for FIFO eviction
	capacity int
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		capacity = 100
	}
	return &Cache[K, V]{
		data:     make(map[K]entry[V]),
		capacity: capacity,
	}
}

// Set stores a value with no expiry.
func (c *Cache[K, V]) Set(key K, value V) {
	c.set(key, entry[V]{value: value})
}

// SetWithTTL stores a value that expires after the given duration.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.set(key, entry[V]{value: value, expiresAt: time.Now().Add(ttl), hasExpiry: true})
}

func (c *Cache[K, V]) set(key K, e entry[V]) {
	_, existed := c.data[key]
	c.data[key] = e
	if !existed {
		c.order = append(c.order, key)
		c.evictIfNeeded()
	}
}

// evictIfNeeded removes the OLDEST entry (FIFO) once capacity is exceeded —
// a simple eviction policy, deliberately not a full LRU, to keep the
// generics the focus rather than the eviction algorithm itself.
func (c *Cache[K, V]) evictIfNeeded() {
	for len(c.data) > c.capacity && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.data, oldest)
	}
}

// Get returns the value and true if the key exists AND hasn't expired.
// A lazily-expired entry is deleted right here, on access — no background
// goroutine needed (that's a later module's territory).
func (c *Cache[K, V]) Get(key K) (V, bool) {
	var zero V
	e, ok := c.data[key]
	if !ok {
		return zero, false
	}
	if e.hasExpiry && time.Now().After(e.expiresAt) {
		c.Delete(key)
		return zero, false
	}
	return e.value, true
}

func (c *Cache[K, V]) Delete(key K) {
	if _, ok := c.data[key]; !ok {
		return
	}
	delete(c.data, key)
	for i, k := range c.order {
		if k == key { // legal because K is `comparable`
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *Cache[K, V]) Len() int {
	return len(c.data)
}

func (c *Cache[K, V]) Keys() []K {
	out := make([]K, len(c.order))
	copy(out, c.order)
	return out
}

// --- Demo -----------------------------------------------------

type User struct {
	Name  string
	Email string
}

func main() {
	fmt.Println("=== Cache[string, User] ===")
	userCache := NewCache[string, User](3)
	userCache.Set("u1", User{Name: "Ada", Email: "ada@example.com"})
	userCache.Set("u2", User{Name: "Kemi", Email: "kemi@example.com"})

	if u, ok := userCache.Get("u1"); ok {
		fmt.Println("found:", u.Name)
	}
	if _, ok := userCache.Get("missing"); !ok {
		fmt.Println("u3 correctly not found")
	}

	fmt.Println("\n=== Capacity eviction (capacity: 3) ===")
	userCache.Set("u3", User{Name: "Tolu", Email: "tolu@example.com"})
	userCache.Set("u4", User{Name: "Zainab", Email: "zainab@example.com"}) // pushes capacity to 4 -> evicts u1
	_, ok := userCache.Get("u1")
	fmt.Println("u1 still present after 4th insert?", ok) // false — evicted, FIFO
	fmt.Println("current keys:", userCache.Keys())

	fmt.Println("\n=== TTL expiry ===")
	sessionCache := NewCache[string, string](10)
	sessionCache.SetWithTTL("session-abc", "logged-in", 50*time.Millisecond)
	val, ok := sessionCache.Get("session-abc")
	fmt.Println("immediately after set:", val, ok)

	time.Sleep(60 * time.Millisecond)
	_, ok = sessionCache.Get("session-abc")
	fmt.Println("after TTL expires:", ok) // false — lazily expired on this Get call

	fmt.Println("\n=== Cache[int, float64] — same implementation, totally different types ===")
	priceCache := NewCache[int, float64](5)
	priceCache.Set(101, 19.99)
	priceCache.Set(102, 45.00)
	price, _ := priceCache.Get(101)
	fmt.Printf("product 101: $%.2f\n", price)
}
