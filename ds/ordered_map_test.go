package ds_test

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/Murlokich/Dixchange/ds"
	"github.com/Murlokich/Dixchange/ds/avltree"
	"github.com/Murlokich/Dixchange/ds/btree"
	"github.com/Murlokich/Dixchange/ds/rbtree"
	"github.com/Murlokich/Dixchange/ds/skiplist"
	"github.com/stretchr/testify/require"
)

// orderedMap is the interface under test, instantiated as the order book uses it.
type orderedMap = ds.OrderedMap[int64, string]

const (
	skipListMaxLevel = 4
	skipListSeed     = 1
)

// Every test runs on each implementation. The skip list runs in both orders, since its head
// end differs: "first" and "last" mean different keys.
//
//nolint:gochecknoglobals // acceptable in test
var implementations = []struct {
	name   string
	create func() orderedMap
}{
	{name: "skiplist ascending", create: func() orderedMap {
		return skiplist.New[int64, string](skipListMaxLevel, skipListSeed, true)
	}},
	{name: "skiplist descending", create: func() orderedMap {
		return skiplist.New[int64, string](skipListMaxLevel, skipListSeed, false)
	}},
	{name: "rbtree", create: func() orderedMap {
		return rbtree.New[int64, string]()
	}},
	{name: "avltree", create: func() orderedMap {
		return avltree.New[int64, string]()
	}},
	{name: "btree t=2", create: func() orderedMap {
		return btree.New[int64, string](2)
	}},
	{name: "btree t=8", create: func() orderedMap {
		return btree.New[int64, string](8)
	}},
}

func newMap(create func() orderedMap, keys ...int64) orderedMap {
	m := create()
	for _, key := range keys {
		m.Put(key, value(key))
	}
	return m
}

func value(key int64) string {
	return "v" + strconv.FormatInt(key, 10)
}

func TestLen(t *testing.T) {
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			require.Equal(t, 0, newMap(impl.create).Len())
			require.Equal(t, 3, newMap(impl.create, 20, 10, 30).Len())
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name   string
		key    int64
		wantOK bool
	}{
		{name: "smallest key", key: 10, wantOK: true},
		{name: "middle key", key: 20, wantOK: true},
		{name: "largest key", key: 30, wantOK: true},
		{name: "missing, below all keys", key: 5, wantOK: false},
		{name: "missing, between keys", key: 15, wantOK: false},
		{name: "missing, above all keys", key: 35, wantOK: false},
	}
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			m := newMap(impl.create, 20, 10, 30)
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, ok := m.Get(tt.key)
					require.Equal(t, tt.wantOK, ok)
					if tt.wantOK {
						require.Equal(t, value(tt.key), got)
					} else {
						require.Empty(t, got)
					}
				})
			}
		})
	}
}

func TestGet_Empty(t *testing.T) {
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			_, ok := newMap(impl.create).Get(10)
			require.False(t, ok)
		})
	}
}

func TestPut_NewKey(t *testing.T) {
	tests := []struct {
		name string
		keys []int64
		key  int64
	}{
		{name: "into empty", keys: nil, key: 10},
		{name: "below all keys", keys: []int64{10, 20, 30}, key: 5},
		{name: "between keys", keys: []int64{10, 20, 30}, key: 15},
		{name: "above all keys", keys: []int64{10, 20, 30}, key: 35},
	}
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					m := newMap(impl.create, tt.keys...)

					m.Put(tt.key, value(tt.key))

					require.Equal(t, len(tt.keys)+1, m.Len())
					for _, key := range slices.Concat(tt.keys, []int64{tt.key}) {
						got, ok := m.Get(key)
						require.True(t, ok, "key %d", key)
						require.Equal(t, value(key), got)
					}
				})
			}
		})
	}
}

func TestPut_ExistingKey(t *testing.T) {
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			m := newMap(impl.create, 10, 20, 30)

			m.Put(20, "replaced")

			require.Equal(t, 3, m.Len())
			got, ok := m.Get(20)
			require.True(t, ok)
			require.Equal(t, "replaced", got)
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name   string
		key    int64
		wantOK bool
	}{
		{name: "smallest key", key: 10, wantOK: true},
		{name: "middle key", key: 20, wantOK: true},
		{name: "largest key", key: 30, wantOK: true},
		{name: "missing key", key: 15, wantOK: false},
	}
	keys := []int64{10, 20, 30}
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					m := newMap(impl.create, keys...)

					require.Equal(t, tt.wantOK, m.Delete(tt.key))

					wantLen := len(keys)
					if tt.wantOK {
						wantLen--
					}
					require.Equal(t, wantLen, m.Len())
					_, ok := m.Get(tt.key)
					require.False(t, ok)
					for _, key := range keys {
						if key == tt.key {
							continue
						}
						_, ok = m.Get(key)
						require.True(t, ok, "key %d", key)
					}
				})
			}
		})
	}
}

func TestDelete_Empty(t *testing.T) {
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			m := newMap(impl.create)
			require.False(t, m.Delete(10))
			require.Equal(t, 0, m.Len())
		})
	}
}

func TestMinMax(t *testing.T) {
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			m := newMap(impl.create)
			_, _, ok := m.Min()
			require.False(t, ok, "min of empty")
			_, _, ok = m.Max()
			require.False(t, ok, "max of empty")

			m.Put(20, value(20))
			requireMinMax(t, m, 20, 20)

			m.Put(30, value(30))
			m.Put(10, value(10))
			requireMinMax(t, m, 10, 30)

			// removing either end moves it to the next key
			m.Delete(10)
			requireMinMax(t, m, 20, 30)
			m.Delete(30)
			requireMinMax(t, m, 20, 20)

			m.Delete(20)
			_, _, ok = m.Min()
			require.False(t, ok, "min after deleting all")
			_, _, ok = m.Max()
			require.False(t, ok, "max after deleting all")
		})
	}
}

func requireMinMax(t *testing.T, m orderedMap, wantMin, wantMax int64) {
	t.Helper()
	key, val, ok := m.Min()
	require.True(t, ok)
	require.Equal(t, wantMin, key)
	require.Equal(t, value(wantMin), val)
	key, val, ok = m.Max()
	require.True(t, ok)
	require.Equal(t, wantMax, key)
	require.Equal(t, value(wantMax), val)
}

// With a few keys the structures stay shallow; enough keys use all the skip list levels and
// make the tree rebalance.
func TestManyKeys(t *testing.T) {
	const n = 1000
	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			keys := rand.New(rand.NewPCG(1, 2)).Perm(n)
			m := newMap(impl.create)
			for _, key := range keys {
				m.Put(int64(key), value(int64(key)))
			}
			require.Equal(t, n, m.Len())
			requireMinMax(t, m, 0, n-1)

			// delete every other key, the rest must stay intact
			for key := int64(0); key < n; key += 2 {
				require.True(t, m.Delete(key))
			}
			require.Equal(t, n/2, m.Len())
			for key := range int64(n) {
				got, ok := m.Get(key)
				require.Equal(t, key%2 == 1, ok, "key %d", key)
				if ok {
					require.Equal(t, value(key), got)
				}
			}
			requireMinMax(t, m, 1, n-1)
		})
	}
}
