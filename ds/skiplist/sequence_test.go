package skiplist_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/Murlokich/Dixchange/ds/skiplist"
	"github.com/stretchr/testify/require"
)

// All keys used by the sequences lie in [0, maxKey]; check also reads the keys just outside.
const maxKey = 41

// sequence keeps a plain Go map next to the skip list and checks after every step that the
// skip list holds exactly the same keys and values.
type sequence struct {
	t   *testing.T
	m   *skiplist.SkipList[int64, string]
	ref map[int64]string
}

func newSequence(t *testing.T, asc bool) *sequence {
	t.Helper()
	s := &sequence{t: t, m: newSkipList(asc), ref: map[int64]string{}}
	s.check()
	return s
}

func (s *sequence) put(key int64, val string) {
	s.t.Helper()
	s.m.Put(key, val)
	s.ref[key] = val
	s.check()
}

func (s *sequence) putAll(keys ...int64) {
	s.t.Helper()
	for _, key := range keys {
		s.put(key, value(key))
	}
}

func (s *sequence) delete(key int64) {
	s.t.Helper()
	_, want := s.ref[key]
	require.Equal(s.t, want, s.m.Delete(key), "delete %d", key)
	delete(s.ref, key)
	s.check()
}

func (s *sequence) deleteAll(keys ...int64) {
	s.t.Helper()
	for _, key := range keys {
		s.delete(key)
	}
}

func (s *sequence) check() {
	s.t.Helper()
	require.Equal(s.t, len(s.ref), s.m.Len())
	for key := int64(-1); key <= maxKey+1; key++ {
		got, ok := s.m.Get(key)
		want, wantOK := s.ref[key]
		require.Equal(s.t, wantOK, ok, "get %d", key)
		require.Equal(s.t, want, got, "get %d", key)
	}

	gotMin, minVal, minOK := s.m.Min()
	gotMax, maxVal, maxOK := s.m.Max()
	if len(s.ref) == 0 {
		require.False(s.t, minOK, "min of empty")
		require.False(s.t, maxOK, "max of empty")
		return
	}
	keys := slices.Sorted(maps.Keys(s.ref))
	wantMin, wantMax := keys[0], keys[len(keys)-1]
	require.True(s.t, minOK)
	require.Equal(s.t, wantMin, gotMin)
	require.Equal(s.t, s.ref[wantMin], minVal)
	require.True(s.t, maxOK)
	require.Equal(s.t, wantMax, gotMax)
	require.Equal(s.t, s.ref[wantMax], maxVal)
}

// Fill with 20 keys in mixed order, drain to empty in another order, then use the map again.
func TestSequence_FillAndDrain(t *testing.T) {
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			s := newSequence(t, order.asc)
			s.putAll(7, 15, 2, 19, 11, 4, 18, 1, 13, 9, 20, 6, 16, 3, 12, 8, 17, 5, 14, 10)
			s.deleteAll(11, 1, 20, 7, 15, 2, 19, 4, 18, 13, 9, 6, 16, 3, 12, 8, 17, 5, 14, 10)
			s.putAll(12, 3, 30)
		})
	}
}

// Order book style: 20 price levels, the levels at both ends are consumed and new ones appear
// beyond the ends and in the gaps.
func TestSequence_BestLevelChurn(t *testing.T) {
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			s := newSequence(t, order.asc)
			s.putAll(20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39)

			s.deleteAll(20, 21, 22)  // lowest levels consumed
			s.putAll(19, 18)         // new levels below the lowest
			s.deleteAll(18, 19, 23)  // consumed again
			s.deleteAll(39, 38, 37)  // highest levels consumed
			s.putAll(41, 40)         // new levels above the highest
			s.deleteAll(30, 32, 34)  // levels in the middle emptied by cancels
			s.putAll(32, 30)         // and recreated
			s.put(25, "replaced 25") // existing level gets a new value
			s.put(41, "replaced 41")
			s.deleteAll(24, 25, 26, 27, 28, 29, 41, 40)
		})
	}
}

// Overwrites and deletes of missing keys must not change anything else.
func TestSequence_ReplaceAndMissing(t *testing.T) {
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			s := newSequence(t, order.asc)
			s.putAll(19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0)

			for key := int64(0); key < 20; key += 3 {
				s.put(key, "replaced")
			}
			s.deleteAll(20, 21, 40) // missing
			for key := int64(0); key < 20; key += 2 {
				s.delete(key)
			}
			s.deleteAll(0, 2, 4) // already deleted
			for key := int64(0); key < 20; key += 2 {
				s.put(key, "back")
			}
		})
	}
}
