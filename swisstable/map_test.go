package swisstable

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestEmpty(t *testing.T) {
	m := New[int, int]()
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
	m := New[string, int]()
	it, isNew := m.Insert("a", 1)
	if !isNew || !it.IsValid() || it.Key() != "a" || it.Value() != 1 {
		t.Fatalf("first Insert: isNew=%v valid=%v key=%v v=%v", isNew, it.IsValid(), it.Key(), it.Value())
	}
	// Strict insert is a no-op on duplicate; returned iterator points at the existing entry.
	it, isNew = m.Insert("a", 999)
	if isNew {
		t.Fatalf("duplicate Insert should report isNew=false")
	}
	if it.Value() != 1 {
		t.Fatalf("Insert must not overwrite existing value; got %d, want 1", it.Value())
	}
	if m.Size() != 1 {
		t.Fatalf("size=%d", m.Size())
	}
}

func TestInsertOrAssign(t *testing.T) {
	m := New[string, int]()
	if old, replaced := m.InsertOrAssign("a", 1); replaced || old != 0 {
		t.Fatalf("first InsertOrAssign: old=%v replaced=%v", old, replaced)
	}
	if it := m.Find("a"); !it.IsValid() || it.Value() != 1 {
		t.Fatalf("Find a: valid=%v v=%v", it.IsValid(), it.Value())
	}
	if old, replaced := m.InsertOrAssign("a", 2); !replaced || old != 1 {
		t.Fatalf("replace InsertOrAssign: old=%v replaced=%v", old, replaced)
	}
	if v := m.Find("a").Value(); v != 2 {
		t.Fatalf("after replace: v=%v", v)
	}
	if m.Size() != 1 {
		t.Fatalf("size=%d", m.Size())
	}
}

func TestErase(t *testing.T) {
	m := New[int, int]()
	for i := 0; i < 100; i++ {
		m.Insert(i, i*10)
	}
	if m.Size() != 100 {
		t.Fatalf("size=%d", m.Size())
	}
	for i := 0; i < 100; i += 2 {
		old, ok := m.Erase(i)
		if !ok || old != i*10 {
			t.Fatalf("Erase %d: old=%v ok=%v", i, old, ok)
		}
	}
	if m.Size() != 50 {
		t.Fatalf("size after erase=%d", m.Size())
	}
	for i := 0; i < 100; i++ {
		it := m.Find(i)
		wantPresent := i%2 == 1
		if it.IsValid() != wantPresent {
			t.Fatalf("Find %d: valid=%v want=%v", i, it.IsValid(), wantPresent)
		}
		if wantPresent && it.Value() != i*10 {
			t.Fatalf("Find %d: v=%v", i, it.Value())
		}
	}
}

func TestGrowMany(t *testing.T) {
	const N = 100_000
	m := New[int, int]()
	for i := 0; i < N; i++ {
		m.Insert(i, -i)
	}
	if m.Size() != N {
		t.Fatalf("size=%d", m.Size())
	}
	for i := 0; i < N; i++ {
		it := m.Find(i)
		if !it.IsValid() || it.Value() != -i {
			t.Fatalf("Find %d: valid=%v v=%v", i, it.IsValid(), it.Value())
		}
	}
	if m.Find(N + 1).IsValid() {
		t.Fatal("Find of absent returned valid iterator")
	}
}

func TestIterateCoversAll(t *testing.T) {
	const N = 5000
	m := New[int, string]()
	want := make(map[int]string, N)
	for i := 0; i < N; i++ {
		s := fmt.Sprintf("v%d", i)
		m.Insert(i, s)
		want[i] = s
	}

	seen := make(map[int]string, N)
	for it := m.Begin(); it.IsValid(); it = it.Next() {
		k, v := it.Key(), it.Value()
		if _, dup := seen[k]; dup {
			t.Fatalf("duplicate key in iteration: %d", k)
		}
		seen[k] = v
	}
	if len(seen) != N {
		t.Fatalf("iterated %d of %d", len(seen), N)
	}
	for k, v := range want {
		if got := seen[k]; got != v {
			t.Fatalf("k=%d: got %q want %q", k, got, v)
		}
	}

	seen2 := make(map[int]string, N)
	for k, v := range m.All() {
		seen2[k] = v
	}
	if len(seen2) != N {
		t.Fatalf("All iterated %d of %d", len(seen2), N)
	}
}

func TestTombstoneReuseAndRehash(t *testing.T) {
	m := New[int, int]()
	// Fill, erase half, refill — exercises tombstone path and same-size rehash.
	for i := 0; i < 1000; i++ {
		m.Insert(i, i)
	}
	for i := 0; i < 1000; i += 2 {
		m.Erase(i)
	}
	for i := 1000; i < 2000; i++ {
		m.Insert(i, i)
	}
	for i := 0; i < 2000; i++ {
		it := m.Find(i)
		wantPresent := i%2 == 1 || i >= 1000
		if it.IsValid() != wantPresent {
			t.Fatalf("i=%d valid=%v want=%v", i, it.IsValid(), wantPresent)
		}
		if wantPresent && it.Value() != i {
			t.Fatalf("i=%d v=%v", i, it.Value())
		}
	}
}

func TestRandomMix(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	m := New[uint64, uint64]()
	ref := make(map[uint64]uint64)
	for i := 0; i < 50_000; i++ {
		k := rng.Uint64() & 0xffff
		switch rng.IntN(3) {
		case 0, 1: // upsert
			v := rng.Uint64()
			m.InsertOrAssign(k, v)
			ref[k] = v
		case 2: // erase
			m.Erase(k)
			delete(ref, k)
		}
	}
	if m.Size() != len(ref) {
		t.Fatalf("size: %d vs ref %d", m.Size(), len(ref))
	}
	for k, v := range ref {
		it := m.Find(k)
		if !it.IsValid() || it.Value() != v {
			t.Fatalf("k=%d: valid=%v v=%v want=%v", k, it.IsValid(), it.Value(), v)
		}
	}
	seen := 0
	for k, v := range m.All() {
		got, ok := ref[k]
		if !ok || got != v {
			t.Fatalf("All k=%d v=%v not in ref or mismatched (ref=%v,%v)", k, v, got, ok)
		}
		seen++
	}
	if seen != len(ref) {
		t.Fatalf("All saw %d, want %d", seen, len(ref))
	}
}

func TestReserveNoRehash(t *testing.T) {
	m := NewWithCapacity[int, int](10_000)
	cap0 := len(m.groups)
	for i := 0; i < 8_000; i++ {
		m.Insert(i, i)
	}
	if len(m.groups) != cap0 {
		t.Fatalf("rehashed despite reserve: %d -> %d groups", cap0, len(m.groups))
	}
}

func TestClear(t *testing.T) {
	m := New[int, int]()
	for i := 0; i < 100; i++ {
		m.Insert(i, i)
	}
	m.Clear()
	if !m.Empty() || m.Size() != 0 {
		t.Fatalf("not empty after Clear")
	}
	if m.Begin().IsValid() {
		t.Fatal("Begin valid after Clear")
	}
	// Reusable after Clear.
	m.Insert(5, 5)
	if m.Size() != 1 {
		t.Fatalf("size=%d", m.Size())
	}
	if it := m.Find(5); !it.IsValid() || it.Value() != 5 {
		t.Fatalf("Find after Clear: valid=%v v=%v", it.IsValid(), it.Value())
	}
}

// Struct keys exercise maphash.Comparable on composite types.
type compKey struct {
	A string
	B int
}

func TestStructKey(t *testing.T) {
	m := New[compKey, int]()
	for i := 0; i < 200; i++ {
		m.Insert(compKey{A: fmt.Sprintf("s%d", i%17), B: i}, i)
	}
	if m.Size() != 200 {
		t.Fatalf("size=%d", m.Size())
	}
	for i := 0; i < 200; i++ {
		it := m.Find(compKey{A: fmt.Sprintf("s%d", i%17), B: i})
		if !it.IsValid() || it.Value() != i {
			t.Fatalf("i=%d valid=%v v=%v", i, it.IsValid(), it.Value())
		}
	}
}
