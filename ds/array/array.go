// Package array implements an ordered map over int64 keys in a fixed range as an array with
// one slot per tick: a key's slot is computed from the key, and a bitset over the slots marks
// which ones are occupied.
package array

import (
	"fmt"
	"math/bits"
)

const wordBits = 64

// Array must be created with New; the zero value is not usable.
type Array[V any] struct {
	minKey int64
	maxKey int64
	tick   int64
	slots  []V
	// occupied slots: bit i%wordBits of set[i/wordBits] is 1 when slots[i] holds a value
	set    []uint64
	length int
	first  int // slot of the smallest key, -1 when empty
	last   int // slot of the largest key, -1 when empty
}

// New creates an empty array for the keys minKey, minKey+tick, ..., maxKey. Other keys are
// rejected: Put panics on them, Get and Delete report them as missing.
func New[V any](minKey, maxKey, tick int64) *Array[V] {
	if tick <= 0 || minKey > maxKey || (maxKey-minKey)%tick != 0 {
		panic(fmt.Sprintf("array: invalid range [%v, %v] with tick %v", minKey, maxKey, tick))
	}
	size := int((maxKey-minKey)/tick) + 1
	return &Array[V]{
		minKey: minKey,
		maxKey: maxKey,
		tick:   tick,
		slots:  make([]V, size),
		set:    make([]uint64, (size+wordBits-1)/wordBits),
		first:  -1,
		last:   -1,
	}
}

func (a *Array[V]) Len() int {
	return a.length
}

func (a *Array[V]) Get(key int64) (V, bool) {
	i, ok := a.slot(key)
	if !ok || !a.occupied(i) {
		var zero V
		return zero, false
	}
	return a.slots[i], true
}

// Put inserts the key, or replaces its value if it is already in the array. It panics if the
// key is outside the range or not on a tick.
func (a *Array[V]) Put(key int64, val V) {
	i, ok := a.slot(key)
	if !ok {
		panic(fmt.Sprintf("array: key %v outside [%v, %v] or not on tick %v",
			key, a.minKey, a.maxKey, a.tick))
	}
	a.slots[i] = val
	if a.occupied(i) {
		return
	}

	a.set[i/wordBits] |= 1 << (i % wordBits)
	a.length++
	if a.length == 1 {
		a.first, a.last = i, i
		return
	}
	a.first = min(a.first, i)
	a.last = max(a.last, i)
}

// Delete removes the key; false if it wasn't in the array.
func (a *Array[V]) Delete(key int64) bool {
	i, ok := a.slot(key)
	if !ok || !a.occupied(i) {
		return false
	}

	var zero V
	a.slots[i] = zero
	a.set[i/wordBits] &^= 1 << (i % wordBits)
	a.length--
	if a.length == 0 {
		a.first, a.last = -1, -1
		return true
	}
	if i == a.first {
		a.first = a.nextOccupied(i + 1)
	}
	if i == a.last {
		a.last = a.prevOccupied(i - 1)
	}
	return true
}

// Min returns the smallest key and its value; false if the array is empty.
func (a *Array[V]) Min() (int64, V, bool) {
	return a.entry(a.first)
}

// Max returns the largest key and its value; false if the array is empty.
func (a *Array[V]) Max() (int64, V, bool) {
	return a.entry(a.last)
}

// slot returns the index of key's slot; false if the key is outside the range or not on a tick.
func (a *Array[V]) slot(key int64) (int, bool) {
	if key < a.minKey || key > a.maxKey {
		return 0, false
	}
	offset := key - a.minKey
	if offset%a.tick != 0 {
		return 0, false
	}
	return int(offset / a.tick), true
}

func (a *Array[V]) occupied(i int) bool {
	return a.set[i/wordBits]&(1<<(i%wordBits)) != 0
}

// nextOccupied returns the first occupied slot at index i or above, -1 if there is none.
// It checks a whole word of the bitset at a time.
func (a *Array[V]) nextOccupied(i int) int {
	w := i / wordBits
	if w >= len(a.set) {
		return -1
	}
	word := a.set[w] & (^uint64(0) << (i % wordBits)) // drop the slots below i
	for word == 0 {
		w++
		if w == len(a.set) {
			return -1
		}
		word = a.set[w]
	}
	return w*wordBits + bits.TrailingZeros64(word)
}

// prevOccupied returns the last occupied slot at index i or below, -1 if there is none.
// It checks a whole word of the bitset at a time.
func (a *Array[V]) prevOccupied(i int) int {
	if i < 0 {
		return -1
	}
	w := i / wordBits
	word := a.set[w] & (^uint64(0) >> (wordBits - 1 - i%wordBits)) // drop the slots above i
	for word == 0 {
		w--
		if w < 0 {
			return -1
		}
		word = a.set[w]
	}
	return w*wordBits + wordBits - 1 - bits.LeadingZeros64(word)
}

func (a *Array[V]) entry(i int) (int64, V, bool) {
	if i < 0 {
		var zero V
		return 0, zero, false
	}
	return a.minKey + int64(i)*a.tick, a.slots[i], true
}
