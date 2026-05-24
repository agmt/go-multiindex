package rbtree

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

func newIntTree() *Tree[int, int] {
	return New[int, int](cmp.Compare[int])
}

func TestEmpty(t *testing.T) {
	tr := newIntTree()
	if !tr.Empty() {
		t.Fatal("new tree should be empty")
	}
	if tr.Size() != 0 {
		t.Fatalf("size=%d", tr.Size())
	}
	if it := tr.Begin(); it.IsValid() {
		t.Fatal("Begin on empty should be invalid")
	}
	if it := tr.RBegin(); it.IsValid() {
		t.Fatal("RBegin on empty should be invalid")
	}
	if it := tr.Find(42); it.IsValid() {
		t.Fatal("Find on empty should be invalid")
	}
	if err := tr.verify(); err != nil {
		t.Fatalf("invariants violated on empty: %v", err)
	}
}

func TestInsertFindErase(t *testing.T) {
	tr := newIntTree()
	for i := 1; i <= 10; i++ {
		it := tr.Insert(i, i*10)
		if !it.IsValid() || it.Key() != i || it.Value() != i*10 {
			t.Fatalf("insert %d: bad iterator", i)
		}
	}
	if tr.Size() != 10 {
		t.Fatalf("size=%d", tr.Size())
	}
	for i := 1; i <= 10; i++ {
		it := tr.Find(i)
		if !it.IsValid() || it.Value() != i*10 {
			t.Fatalf("Find %d: invalid or wrong value", i)
		}
	}
	if it := tr.Find(99); it.IsValid() {
		t.Fatal("Find 99 should be invalid")
	}
	tr.Erase(tr.Find(5))
	if tr.Size() != 9 {
		t.Fatalf("after erase, size=%d", tr.Size())
	}
	if it := tr.Find(5); it.IsValid() {
		t.Fatal("erased key still found")
	}
	if err := tr.verify(); err != nil {
		t.Fatalf("invariants violated: %v", err)
	}
}

func TestDuplicateKeys(t *testing.T) {
	tr := newIntTree()
	for i := 0; i < 5; i++ {
		tr.Insert(7, i)
	}
	if tr.Size() != 5 {
		t.Fatalf("size=%d", tr.Size())
	}
	count := 0
	for it := tr.LowerBound(7); it.IsValid() && it.Key() == 7; it = it.Next() {
		count++
	}
	if count != 5 {
		t.Fatalf("walked %d duplicates, want 5", count)
	}
	// Erase all of them.
	for it := tr.Find(7); it.IsValid(); it = tr.Find(7) {
		tr.Erase(it)
	}
	if !tr.Empty() {
		t.Fatalf("not empty after erasing all 7s: size=%d", tr.Size())
	}
}

func TestOrderedTraversal(t *testing.T) {
	tr := newIntTree()
	keys := []int{5, 2, 8, 1, 9, 3, 7, 4, 6, 0}
	for _, k := range keys {
		tr.Insert(k, k*10)
	}
	got := []int{}
	tr.Traversal(func(k, v int) bool {
		got = append(got, k)
		return true
	})
	want := slices.Clone(keys)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("Traversal order:\n got=%v\nwant=%v", got, want)
	}

	got2 := []int{}
	for k := range tr.All() {
		got2 = append(got2, k)
	}
	if !slices.Equal(got2, want) {
		t.Fatalf("All order:\n got=%v\nwant=%v", got2, want)
	}
}

func TestBidirectionalIteration(t *testing.T) {
	tr := newIntTree()
	for _, k := range []int{3, 1, 4, 1, 5, 9, 2, 6, 5} {
		tr.Insert(k, k)
	}
	forward := []int{}
	for it := tr.Begin(); it.IsValid(); it = it.Next() {
		forward = append(forward, it.Key())
	}
	backward := []int{}
	for it := tr.RBegin(); it.IsValid(); it = it.Prev() {
		backward = append(backward, it.Key())
	}
	rev := slices.Clone(forward)
	slices.Reverse(rev)
	if !slices.Equal(rev, backward) {
		t.Fatalf("Prev not symmetric with Next:\nfwd  =%v\nback =%v\nrev fwd=%v",
			forward, backward, rev)
	}
}

func TestLowerUpperBound(t *testing.T) {
	tr := newIntTree()
	for _, k := range []int{10, 20, 30, 30, 40, 50} {
		tr.Insert(k, k)
	}
	cases := []struct {
		key                 int
		wantLower, wantUpper int
		validLower, validUpper bool
	}{
		{5, 10, 10, true, true},
		{10, 10, 20, true, true},
		{25, 30, 30, true, true},
		{30, 30, 40, true, true},
		{50, 50, 0, true, false},
		{99, 0, 0, false, false},
	}
	for _, c := range cases {
		lb := tr.LowerBound(c.key)
		if lb.IsValid() != c.validLower {
			t.Errorf("LowerBound(%d).IsValid=%v want %v", c.key, lb.IsValid(), c.validLower)
		}
		if c.validLower && lb.Key() != c.wantLower {
			t.Errorf("LowerBound(%d)=%d want %d", c.key, lb.Key(), c.wantLower)
		}
		ub := tr.UpperBound(c.key)
		if ub.IsValid() != c.validUpper {
			t.Errorf("UpperBound(%d).IsValid=%v want %v", c.key, ub.IsValid(), c.validUpper)
		}
		if c.validUpper && ub.Key() != c.wantUpper {
			t.Errorf("UpperBound(%d)=%d want %d", c.key, ub.Key(), c.wantUpper)
		}
	}
}

func TestSetValue(t *testing.T) {
	tr := newIntTree()
	tr.Insert(1, 100)
	it := tr.Find(1)
	it.SetValue(999)
	if v := tr.Find(1).Value(); v != 999 {
		t.Fatalf("SetValue did not persist: got %d", v)
	}
}

func TestClear(t *testing.T) {
	tr := newIntTree()
	for i := 0; i < 100; i++ {
		tr.Insert(i, i)
	}
	tr.Clear()
	if !tr.Empty() || tr.Size() != 0 {
		t.Fatalf("not empty after Clear")
	}
	if it := tr.Begin(); it.IsValid() {
		t.Fatal("Begin valid after Clear")
	}
	// Reusable after Clear.
	tr.Insert(5, 5)
	if tr.Size() != 1 || tr.Find(5).Value() != 5 {
		t.Fatal("Insert after Clear broken")
	}
}

func TestCustomComparator(t *testing.T) {
	// Reverse order.
	tr := New[int, int](func(a, b int) int { return cmp.Compare(b, a) })
	for _, k := range []int{1, 5, 3, 2, 4} {
		tr.Insert(k, k)
	}
	got := []int{}
	tr.Traversal(func(k, v int) bool {
		got = append(got, k)
		return true
	})
	want := []int{5, 4, 3, 2, 1}
	if !slices.Equal(got, want) {
		t.Fatalf("reverse-ordered traversal: got=%v want=%v", got, want)
	}
}

func TestRandomMixVsRefMap(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 22))
	tr := newIntTree()
	ref := make(map[int]int)
	for i := 0; i < 50_000; i++ {
		k := rng.IntN(2048)
		switch rng.IntN(3) {
		case 0, 1:
			v := rng.Int()
			// Tree allows duplicates; mirror only the latest mapping by erasing then inserting.
			if it := tr.Find(k); it.IsValid() {
				tr.Erase(it)
			}
			tr.Insert(k, v)
			ref[k] = v
		case 2:
			if it := tr.Find(k); it.IsValid() {
				tr.Erase(it)
				delete(ref, k)
			}
		}
		if i%1000 == 0 {
			if err := tr.verify(); err != nil {
				t.Fatalf("iter %d: invariants: %v", i, err)
			}
		}
	}
	if tr.Size() != len(ref) {
		t.Fatalf("size: tr=%d ref=%d", tr.Size(), len(ref))
	}
	// Membership + value parity.
	for k, v := range ref {
		got := tr.Find(k)
		if !got.IsValid() || got.Value() != v {
			t.Fatalf("k=%d: got=(%v,%v) want=%v", k, got.IsValid(), got.Value(), v)
		}
	}
	// Sorted iteration parity.
	wantKeys := make([]int, 0, len(ref))
	for k := range ref {
		wantKeys = append(wantKeys, k)
	}
	slices.Sort(wantKeys)
	gotKeys := []int{}
	for it := tr.Begin(); it.IsValid(); it = it.Next() {
		gotKeys = append(gotKeys, it.Key())
	}
	if !slices.Equal(gotKeys, wantKeys) {
		t.Fatalf("sorted iteration mismatch")
	}
	if err := tr.verify(); err != nil {
		t.Fatalf("final invariants: %v", err)
	}
}

type compKey struct {
	A string
	B int
}

func TestStructKey(t *testing.T) {
	tr := New[compKey, int](func(a, b compKey) int {
		if c := cmp.Compare(a.A, b.A); c != 0 {
			return c
		}
		return cmp.Compare(a.B, b.B)
	})
	for i := 0; i < 50; i++ {
		tr.Insert(compKey{A: fmt.Sprintf("s%d", i%7), B: i}, i)
	}
	if tr.Size() != 50 {
		t.Fatalf("size=%d", tr.Size())
	}
	for i := 0; i < 50; i++ {
		it := tr.Find(compKey{A: fmt.Sprintf("s%d", i%7), B: i})
		if !it.IsValid() || it.Value() != i {
			t.Fatalf("i=%d: invalid or wrong value", i)
		}
	}
	if err := tr.verify(); err != nil {
		t.Fatalf("invariants: %v", err)
	}
}

func TestIteratorEqual(t *testing.T) {
	tr := newIntTree()
	tr.Insert(1, 1)
	tr.Insert(2, 2)
	a := tr.Find(1)
	b := tr.Find(1)
	c := tr.Find(2)
	if !a.Equal(b) {
		t.Fatal("two Find(1) should be Equal")
	}
	if a.Equal(c) {
		t.Fatal("Find(1) and Find(2) should not be Equal")
	}
	end := Iterator[int, int]{}
	if !tr.Find(99).Equal(end) {
		t.Fatal("Find of absent should Equal zero-value (end) iterator")
	}
}
