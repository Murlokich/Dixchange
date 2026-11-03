package ds_test

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"testing"

	gbtree "github.com/google/btree"
)

// With -steps, every fuzz input run is printed as the operations it decodes to. Use it to read
// a saved failing input:
//
//	go test ./ds -run=FuzzOrderedMap/<hash> -v -steps
//
//nolint:gochecknoglobals // test flag
var printSteps = flag.Bool("steps", false, "print the operations of each FuzzOrderedMap input")

type opKind uint8

const (
	opPut opKind = iota
	opDelete
)

const (
	opKindCount = 2 // number of opKinds, for decoding
	opBytes     = 2 // bytes per operation: kind, key
	maxOpKey    = 255
)

type op struct {
	kind opKind
	key  int64
}

func (o op) String() string {
	if o.kind == opPut {
		return fmt.Sprintf("Put(%d)", o.key)
	}
	return fmt.Sprintf("Delete(%d)", o.key)
}

// decodeOps reads the fuzz input two bytes per operation: the first byte picks the kind, the
// second is the key (0..maxOpKey, so always inside every implementation's key range).
// A trailing odd byte is ignored.
func decodeOps(data []byte) []op {
	ops := make([]op, 0, len(data)/opBytes)
	for i := 0; i+opBytes <= len(data); i += opBytes {
		ops = append(ops, op{kind: opKind(data[i] % opKindCount), key: int64(data[i+1])})
	}
	return ops
}

// formatOps lists the operations one per line, numbered from 1.
func formatOps(ops []op) string {
	var b strings.Builder
	for i, o := range ops {
		fmt.Fprintf(&b, "%4d. %s\n", i+1, o)
	}
	return b.String()
}

type entry struct {
	key int64
	val string
}

func entryLess(a, b entry) bool {
	return a.key < b.key
}

// FuzzOrderedMap applies the operations decoded from the input to every implementation and to
// google/btree as the reference, and compares the results after each operation.
//
//	go test ./ds -run='^$' -fuzz=FuzzOrderedMap -fuzztime=60s   # fuzz
//	go test ./ds -run=FuzzOrderedMap/<hash> -v -steps          # read a saved input
//	go test ./ds -run=FuzzOrderedMap/<hash>                    # replay it
//
// A failing input is saved to testdata/fuzz/FuzzOrderedMap; plain go test runs it again.
func FuzzOrderedMap(f *testing.F) {
	for _, seed := range fuzzSeeds() {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		r := newFuzzRun(t, decodeOps(data))
		if *printSteps {
			t.Log("steps:\n" + formatOps(r.ops))
		}
		for step := range r.ops {
			r.apply(step)
			r.checkReads(step)
		}
		r.checkAllKeys()
	})
}

// fuzzRun holds one fuzz input's operations, the reference and the implementations under test.
type fuzzRun struct {
	t    *testing.T
	ops  []op
	ref  *gbtree.BTreeG[entry]
	maps []orderedMap
}

func newFuzzRun(t *testing.T, ops []op) *fuzzRun {
	t.Helper()
	r := &fuzzRun{t: t, ops: ops, ref: gbtree.NewG(2, entryLess)}
	for _, impl := range implementations {
		r.maps = append(r.maps, impl.create())
	}
	return r
}

// failf reports a divergence of implementation i and lists the steps up to the failing one.
func (r *fuzzRun) failf(step, i int, format string, args ...any) {
	r.t.Helper()
	r.t.Fatalf("%s diverged at step %d: %s\nsteps:\n%s", implementations[i].name, step+1,
		fmt.Sprintf(format, args...), formatOps(r.ops[:step+1]))
}

func (r *fuzzRun) apply(step int) {
	r.t.Helper()
	o := r.ops[step]
	switch o.kind {
	case opPut:
		val := strconv.Itoa(step) // a new value on every put, so a lost overwrite shows
		r.ref.ReplaceOrInsert(entry{key: o.key, val: val})
		for _, m := range r.maps {
			m.Put(o.key, val)
		}
	case opDelete:
		_, want := r.ref.Delete(entry{key: o.key})
		for i, m := range r.maps {
			if got := m.Delete(o.key); got != want {
				r.failf(step, i, "Delete(%d) = %v, want %v", o.key, got, want)
			}
		}
	}
}

// checkReads compares Len, Min, Max and Get of the key the step touched.
func (r *fuzzRun) checkReads(step int) {
	r.t.Helper()
	key := r.ops[step].key
	wantMin, wantMinOK := r.ref.Min()
	wantMax, wantMaxOK := r.ref.Max()
	wantGet, wantGetOK := r.ref.Get(entry{key: key})
	for i, m := range r.maps {
		if got := m.Len(); got != r.ref.Len() {
			r.failf(step, i, "Len() = %d, want %d", got, r.ref.Len())
		}
		if k, v, ok := m.Min(); !sameEntry(k, v, ok, wantMin, wantMinOK) {
			r.failf(step, i, "Min() = (%d, %q, %v), want (%d, %q, %v)",
				k, v, ok, wantMin.key, wantMin.val, wantMinOK)
		}
		if k, v, ok := m.Max(); !sameEntry(k, v, ok, wantMax, wantMaxOK) {
			r.failf(step, i, "Max() = (%d, %q, %v), want (%d, %q, %v)",
				k, v, ok, wantMax.key, wantMax.val, wantMaxOK)
		}
		if v, ok := m.Get(key); ok != wantGetOK || v != wantGet.val {
			r.failf(step, i, "Get(%d) = (%q, %v), want (%q, %v)", key, v, ok, wantGet.val, wantGetOK)
		}
	}
}

// checkAllKeys compares Get of every possible key after the last step.
func (r *fuzzRun) checkAllKeys() {
	r.t.Helper()
	if len(r.ops) == 0 {
		return
	}
	last := len(r.ops) - 1
	for key := range int64(maxOpKey + 1) {
		want, wantOK := r.ref.Get(entry{key: key})
		for i, m := range r.maps {
			if v, ok := m.Get(key); ok != wantOK || v != want.val {
				r.failf(last, i, "after all steps: Get(%d) = (%q, %v), want (%q, %v)",
					key, v, ok, want.val, wantOK)
			}
		}
	}
}

func sameEntry(key int64, val string, ok bool, want entry, wantOK bool) bool {
	return ok == wantOK && (!ok || key == want.key && val == want.val)
}

// fuzzSeeds are the inputs the fuzzer starts from; plain go test runs them as regular cases.
func fuzzSeeds() [][]byte {
	encode := func(kind opKind, key int) []byte {
		return []byte{byte(kind), byte(key)}
	}
	var fillAndDrain, interleaved, sameKey []byte
	// fill all keys in ascending order, then delete them from the top
	for key := range maxOpKey + 1 {
		fillAndDrain = append(fillAndDrain, encode(opPut, key)...)
	}
	for key := maxOpKey; key >= 0; key-- {
		fillAndDrain = append(fillAndDrain, encode(opDelete, key)...)
	}
	// put keys from both ends towards the middle, deleting every third one on the way
	for i := range 64 {
		interleaved = append(interleaved, encode(opPut, i)...)
		interleaved = append(interleaved, encode(opPut, maxOpKey-i)...)
		if i%3 == 0 {
			interleaved = append(interleaved, encode(opDelete, i/2)...)
		}
	}
	// overwrite and delete one key repeatedly
	for range 4 {
		sameKey = append(sameKey, encode(opPut, 7)...)
		sameKey = append(sameKey, encode(opPut, 7)...)
		sameKey = append(sameKey, encode(opDelete, 7)...)
		sameKey = append(sameKey, encode(opDelete, 7)...)
	}
	return [][]byte{fillAndDrain, interleaved, sameKey}
}
