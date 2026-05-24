package swissmap

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func intKey(v int) int { return v }

// bookByISBN: typical multiindex use case — key is one field of V.
type bookByISBN struct {
	ISBN, Name, Author string
}

func bookISBN(b bookByISBN) string { return b.ISBN }

func TestEmpty(t *testing.T) {
	m := New[int, int](intKey)
	if m.Size() != 0 {
		t.Fatalf("size=%d, want 0", m.Size())
	}
	if !m.Empty() {
		t.Fatal("new map should be Empty()")
	}
	if m.Find(42).IsValid() {
		t.Fatal("Find on empty returned valid iterator")
	}
	if _, ok := m.Erase(42); ok {
		t.Fatal("Erase on empty returned ok")
	}
	if m.Begin().IsValid() {
		t.Fatal("Begin on empty should not be valid")
	}
	count := 0
	for range m.All() {
		count++
	}
	if count != 0 {
		t.Fatalf("iterated %d on empty", count)
	}
}

func TestInsert(t *testing.T) {
	m := New[int, int](intKey)
	it, isNew := m.Insert(7)
	if !isNew || !it.IsValid() || it.Value() != 7 || it.Key() != 7 {
		t.Fatalf("first Insert: isNew=%v valid=%v v=%v k=%v", isNew, it.IsValid(), it.Value(), it.Key())
	}
	it, isNew = m.Insert(7)
	if isNew {
		t.Fatal("duplicate Insert should report isNew=false")
	}
	if it.Value() != 7 {
		t.Fatalf("Insert must not overwrite; got %v", it.Value())
	}
	if m.Size() != 1 {
		t.Fatalf("size=%d", m.Size())
	}
}

func TestInsertOrAssign_ProbeFields(t *testing.T) {
	// keyOf returns only ISBN; Equal on the stored K==K compares only ISBN.
	m := New[bookByISBN, string](bookISBN)
	first := bookByISBN{ISBN: "x", Name: "n1", Author: "a1"}
	second := bookByISBN{ISBN: "x", Name: "n2", Author: "a2"}

	if old, replaced := m.InsertOrAssign(first); replaced || old.ISBN != "" {
		t.Fatalf("first InsertOrAssign: old=%v replaced=%v", old, replaced)
	}
	old, replaced := m.InsertOrAssign(second)
	if !replaced {
		t.Fatal("second InsertOrAssign should have replaced")
	}
	if old != first {
		t.Fatalf("returned old=%v, want %v", old, first)
	}
	// After replace, lookup by key alone returns the second value.
	it := m.Find("x")
	if !it.IsValid() || it.Value() != second {
		t.Fatalf("Find by key: valid=%v v=%v want %v", it.IsValid(), it.Value(), second)
	}
}

func TestErase(t *testing.T) {
	m := New[int, int](intKey)
	for i := 0; i < 100; i++ {
		m.Insert(i)
	}
	for i := 0; i < 100; i += 2 {
		old, ok := m.Erase(i)
		if !ok || old != i {
			t.Fatalf("Erase %d: old=%v ok=%v", i, old, ok)
		}
	}
	if m.Size() != 50 {
		t.Fatalf("size=%d", m.Size())
	}
	for i := 0; i < 100; i++ {
		want := i%2 == 1
		if m.Contains(i) != want {
			t.Fatalf("Contains %d=%v want %v", i, m.Contains(i), want)
		}
	}
}

func TestGrowMany(t *testing.T) {
	const N = 100_000
	m := New[int, int](intKey)
	for i := 0; i < N; i++ {
		m.Insert(i)
	}
	if m.Size() != N {
		t.Fatalf("size=%d", m.Size())
	}
	for i := 0; i < N; i++ {
		if !m.Find(i).IsValid() {
			t.Fatalf("missing %d", i)
		}
	}
	if m.Find(N + 1).IsValid() {
		t.Fatal("Find of absent returned valid")
	}
}

func TestIterateCoversAll(t *testing.T) {
	const N = 5000
	m := New[int, int](intKey)
	want := make(map[int]bool, N)
	for i := 0; i < N; i++ {
		m.Insert(i)
		want[i] = true
	}

	seen := make(map[int]bool, N)
	for it := m.Begin(); it.IsValid(); it = it.Next() {
		v := it.Value()
		if seen[v] {
			t.Fatalf("duplicate in iteration: %d", v)
		}
		seen[v] = true
	}
	if len(seen) != N {
		t.Fatalf("iterated %d of %d", len(seen), N)
	}

	seen2 := make(map[int]bool, N)
	for v := range m.All() {
		seen2[v] = true
	}
	if len(seen2) != N {
		t.Fatalf("All iterated %d of %d", len(seen2), N)
	}

	seen3 := make(map[int]bool, N)
	for k, v := range m.AllKV() {
		if k != v {
			t.Fatalf("AllKV mismatch: k=%d v=%d", k, v)
		}
		seen3[v] = true
	}
	if len(seen3) != N {
		t.Fatalf("AllKV iterated %d of %d", len(seen3), N)
	}
}

func TestTombstoneReuseAndRehash(t *testing.T) {
	m := New[int, int](intKey)
	for i := 0; i < 1000; i++ {
		m.Insert(i)
	}
	for i := 0; i < 1000; i += 2 {
		m.Erase(i)
	}
	for i := 1000; i < 2000; i++ {
		m.Insert(i)
	}
	for i := 0; i < 2000; i++ {
		want := i%2 == 1 || i >= 1000
		if m.Contains(i) != want {
			t.Fatalf("i=%d contains=%v want=%v", i, m.Contains(i), want)
		}
	}
}

func TestRandomMix(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	m := New[int, int](intKey)
	ref := make(map[int]bool)
	for i := 0; i < 50_000; i++ {
		k := int(rng.Uint64() & 0xffff)
		switch rng.IntN(3) {
		case 0, 1:
			m.Insert(k)
			ref[k] = true
		case 2:
			m.Erase(k)
			delete(ref, k)
		}
	}
	if m.Size() != len(ref) {
		t.Fatalf("size: %d vs ref %d", m.Size(), len(ref))
	}
	for k := range ref {
		if !m.Contains(k) {
			t.Fatalf("missing k=%d", k)
		}
	}
	seen := 0
	for v := range m.All() {
		if !ref[v] {
			t.Fatalf("All saw unexpected %d", v)
		}
		seen++
	}
	if seen != len(ref) {
		t.Fatalf("All saw %d, want %d", seen, len(ref))
	}
}

func TestReserveNoRehash(t *testing.T) {
	m := NewWithCapacity[int, int](intKey, 10_000)
	cap0 := len(m.groups)
	for i := 0; i < 8_000; i++ {
		m.Insert(i)
	}
	if len(m.groups) != cap0 {
		t.Fatalf("rehashed despite reserve: %d -> %d groups", cap0, len(m.groups))
	}
}

func TestClear(t *testing.T) {
	m := New[int, int](intKey)
	for i := 0; i < 100; i++ {
		m.Insert(i)
	}
	m.Clear()
	if !m.Empty() || m.Size() != 0 {
		t.Fatalf("not empty after Clear")
	}
	if m.Begin().IsValid() {
		t.Fatal("Begin valid after Clear")
	}
	m.Insert(5)
	if !m.Contains(5) {
		t.Fatal("Insert after Clear broken")
	}
}

func TestStructKeyExtraction(t *testing.T) {
	m := New[bookByISBN, string](bookISBN)
	for i := 0; i < 200; i++ {
		m.Insert(bookByISBN{ISBN: fmt.Sprintf("isbn-%d", i), Name: "n", Author: "a"})
	}
	if m.Size() != 200 {
		t.Fatalf("size=%d", m.Size())
	}
	// Lookup by key alone — caller doesn't need to construct a full V probe.
	for i := 0; i < 200; i++ {
		key := fmt.Sprintf("isbn-%d", i)
		it := m.Find(key)
		if !it.IsValid() {
			t.Fatalf("missing %s", key)
		}
		if it.Value().ISBN != key {
			t.Fatalf("ISBN mismatch: %v", it.Value())
		}
		if it.Key() != key {
			t.Fatalf("Key mismatch: %v", it.Key())
		}
	}
}
