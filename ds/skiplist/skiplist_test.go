package skiplist_test

import (
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/Murlokich/Dixchange/ds/skiplist"
	"github.com/stretchr/testify/require"
)

const (
	testMaxLevel = 4
	testSeed     = 1
)

// Every test runs on both orders of the list: the head end differs, so "first" and "last"
// mean different keys.
//
//nolint:gochecknoglobals // acceptable in test
var orders = []struct {
	name string
	asc  bool
}{
	{name: "ascending", asc: true},
	{name: "descending", asc: false},
}

func newSkipList(asc bool, keys ...int64) *skiplist.SkipList[int64, string] {
	m := skiplist.New[int64, string](testMaxLevel, testSeed, asc)
	for _, key := range keys {
		m.Put(key, value(key))
	}
	return m
}

func value(key int64) string {
	return "v" + strconv.FormatInt(key, 10)
}

func TestLen(t *testing.T) {
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			require.Equal(t, 0, newSkipList(order.asc).Len())
			require.Equal(t, 3, newSkipList(order.asc, 20, 10, 30).Len())
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
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			m := newSkipList(order.asc, 20, 10, 30)
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
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			_, ok := newSkipList(order.asc).Get(10)
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
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					m := newSkipList(order.asc, tt.keys...)

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
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			m := newSkipList(order.asc, 10, 20, 30)

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
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					m := newSkipList(order.asc, keys...)

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
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			m := newSkipList(order.asc)
			require.False(t, m.Delete(10))
			require.Equal(t, 0, m.Len())
		})
	}
}

func TestMinMax(t *testing.T) {
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			m := newSkipList(order.asc)
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

func requireMinMax(t *testing.T, m *skiplist.SkipList[int64, string], wantMin, wantMax int64) {
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

// With a few keys most towers have height 1; enough keys use all the levels.
func TestManyKeys(t *testing.T) {
	const n = 1000
	for _, order := range orders {
		t.Run(order.name, func(t *testing.T) {
			keys := rand.New(rand.NewPCG(1, 2)).Perm(n)
			m := newSkipList(order.asc)
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
