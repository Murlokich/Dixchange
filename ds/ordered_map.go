package ds

import "cmp"

// OrderedMap is the set of operations every structure in ds provides.
type OrderedMap[K cmp.Ordered, V any] interface {
	Len() int
	Get(key K) (V, bool)
	// Put inserts the key, or replaces its value if it is already in the map.
	Put(key K, val V)
	// Delete removes the key; false if it wasn't in the map.
	Delete(key K) bool
	// Min returns the smallest key and its value; false if the map is empty.
	Min() (K, V, bool)
	// Max returns the largest key and its value; false if the map is empty.
	Max() (K, V, bool)
}
