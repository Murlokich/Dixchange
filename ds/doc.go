// Package ds holds the ordered map implementations benchmarked as the price-level structure of
// the order book. They are independent of the application: each one is a generic type with
// K cmp.Ordered keys and V any values in its own subpackage (ds/btree, ds/skiplist, ...),
// compares keys with the native <, and provides the same operations:
//
//	Len() int
//	Get(K) (V, bool)
//	Put(K, V)
//	Delete(K) bool
//	Min() (K, V, bool)
//	Max() (K, V, bool)
package ds
