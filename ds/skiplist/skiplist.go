// Package skiplist implements an ordered map as a skip list (Pugh, 1990).
package skiplist

import (
	"cmp"
	"math/rand/v2"
)

type node[K cmp.Ordered, V any] struct {
	key  K
	val  V
	next []*node[K, V] // next[i] is the following node on level i; len is the node's height
}

// SkipList must be created with New; the zero value is not usable.
type SkipList[K cmp.Ordered, V any] struct {
	head   *node[K, V] // sentinel with a tower of maxLevel pointers, holds no key
	tail   *node[K, V] // last node on level 0, nil when empty
	level  int         // number of levels in use
	length int
	asc    bool
	rng    *rand.PCG
	// predecessors of the key on each level, filled by findPredecessors;
	// kept in the SkipList so Put and Delete don't allocate it
	update []*node[K, V]
}

// New creates an empty skip list. maxLevel caps the height of the towers: with p = 1/4 it
// should be about log4 of the expected number of keys. The seed makes the heights, and so the
// shape of the list, reproducible.
func New[K cmp.Ordered, V any](maxLevel int, seed uint64, asc bool) *SkipList[K, V] {
	if maxLevel < 1 {
		panic("skiplist: maxLevel must be at least 1")
	}
	return &SkipList[K, V]{
		head:   &node[K, V]{next: make([]*node[K, V], maxLevel)},
		asc:    asc,
		rng:    rand.NewPCG(seed, seed),
		update: make([]*node[K, V], maxLevel),
	}
}

func (s *SkipList[K, V]) Len() int {
	return s.length
}

func (s *SkipList[K, V]) Get(key K) (V, bool) {
	x := s.head
	for i := s.level - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && s.before(next.key, key); next = x.next[i] {
			x = next
		}
	}
	x = x.next[0]
	if x == nil || x.key != key {
		var zero V
		return zero, false
	}
	return x.val, true
}

// Put inserts the key, or replaces its value if it is already in the list.
func (s *SkipList[K, V]) Put(key K, val V) {
	x := s.findPredecessors(key)
	if x != nil && x.key == key {
		x.val = val
		return
	}

	height := s.randomHeight()
	for i := s.level; i < height; i++ {
		s.update[i] = s.head
	}
	s.level = max(s.level, height)

	n := &node[K, V]{key: key, val: val, next: make([]*node[K, V], height)}
	for i := range height {
		n.next[i] = s.update[i].next[i]
		s.update[i].next[i] = n
	}
	if n.next[0] == nil {
		s.tail = n
	}
	s.length++
}

// Delete removes the key; false if it wasn't in the list.
func (s *SkipList[K, V]) Delete(key K) bool {
	x := s.findPredecessors(key)
	if x == nil || x.key != key {
		return false
	}

	// on every level of x's tower its predecessor is the one found by the search
	for i := range x.next {
		s.update[i].next[i] = x.next[i]
	}
	if s.tail == x {
		s.tail = s.update[0]
		if s.tail == s.head {
			s.tail = nil
		}
	}
	for s.level > 0 && s.head.next[s.level-1] == nil {
		s.level--
	}
	s.length--
	return true
}

// Min returns the smallest key and its value; false if the list is empty.
func (s *SkipList[K, V]) Min() (K, V, bool) {
	if s.asc {
		return entry(s.head.next[0])
	}
	return entry(s.tail)
}

// Max returns the largest key and its value; false if the list is empty.
func (s *SkipList[K, V]) Max() (K, V, bool) {
	if s.asc {
		return entry(s.tail)
	}
	return entry(s.head.next[0])
}

// findPredecessors fills s.update with the last node before key on each level in use and
// returns the first node not before key (nil if there is none).
func (s *SkipList[K, V]) findPredecessors(key K) *node[K, V] {
	x := s.head
	for i := s.level - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && s.before(next.key, key); next = x.next[i] {
			x = next
		}
		s.update[i] = x
	}
	return x.next[0]
}

// before reports whether a comes before b in the order of the list.
func (s *SkipList[K, V]) before(a, b K) bool {
	if s.asc {
		return a < b
	}
	return a > b
}

// Each extra level of a tower is kept with probability 1/levelOdds.
const levelOdds = 4

// randomHeight draws a tower height: every node gets one level, and each further level is
// added with probability 1/levelOdds, until the first miss or maxLevel.
func (s *SkipList[K, V]) randomHeight() int {
	height := 1
	for height < len(s.head.next) && s.rng.Uint64()%levelOdds == 0 {
		height++
	}
	return height
}

func entry[K cmp.Ordered, V any](n *node[K, V]) (K, V, bool) {
	if n == nil {
		var key K
		var val V
		return key, val, false
	}
	return n.key, n.val, true
}
