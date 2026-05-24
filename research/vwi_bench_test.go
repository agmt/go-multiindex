// Package research benchmarks the cost of "VWI" (value-with-index) container
// designs vs the current "K stored separately" design.
//
// We compare two configurations of the existing rbtree and swisstable APIs:
//
//   - K-stored:  Container[string, Book] with cmp/hash operating on string.
//     Each node/slot stores (K, V) = (string, Book).
//
//   - V-only:    Container[Book, struct{}] with cmp/hash operating on a
//     closure that extracts Book.ISBN. Each node/slot stores
//     just Book (struct{} is zero-sized). This is what a real
//     Tree[V] / Map[V] would store, but built with the
//     existing 2-type-param API.
//
// Caveat: the swisstable V-only sim uses Go's builtin == on the full Book
// struct, comparing all 3 string fields. A real Map[V] with a closure-based
// equal would only compare .ISBN, so swiss V-only is pessimistic. rbtree
// V-only is accurate because the closure-cmp already only reads .ISBN.
package research

import (
	"cmp"
	"fmt"
	"testing"

	"github.com/agmt/go-multiindex/rbtree"
	"github.com/agmt/go-multiindex/swisstable"
)

type Book struct {
	Name, Author, ISBN string
}

var sizes = []int{1_000, 100_000}

func makeBooks(n int) []Book {
	books := make([]Book, n)
	for i := range books {
		books[i] = Book{
			ISBN:   fmt.Sprintf("978-%013d", i),
			Name:   "Some Title That Is Reasonable In Length",
			Author: "Author Name",
		}
	}
	return books
}

// ---- micro: per-call compare / hash cost in isolation ----

var (
	bookA = Book{ISBN: "978-0000000000001", Name: "x", Author: "y"}
	bookB = Book{ISBN: "978-0000000000002", Name: "x", Author: "y"}
)

func BenchmarkMicroCmp_StringDirect(b *testing.B) {
	a, c := bookA.ISBN, bookB.ISBN
	var x int
	for b.Loop() {
		x ^= cmp.Compare(a, c)
	}
	_ = x
}

func BenchmarkMicroCmp_BookViaClosure(b *testing.B) {
	less := func(a, b Book) int { return cmp.Compare(a.ISBN, b.ISBN) }
	var x int
	for b.Loop() {
		x ^= less(bookA, bookB)
	}
	_ = x
}

// Interface-dispatch alternative (option 3 in the design space).
type cmpVWI interface{ Cmp(o cmpVWI) int }
type bookVWI Book

func (a bookVWI) Cmp(o cmpVWI) int { return cmp.Compare(a.ISBN, o.(bookVWI).ISBN) }

func BenchmarkMicroCmp_BookViaInterface(b *testing.B) {
	var ca cmpVWI = bookVWI(bookA)
	var cb cmpVWI = bookVWI(bookB)
	var x int
	for b.Loop() {
		x ^= ca.Cmp(cb)
	}
	_ = x
}

// ---- rbtree: K-stored vs V-only Insert/Find ----

func benchRBTreeInsertKStored(b *testing.B, n int) {
	books := makeBooks(n)
	b.ResetTimer()
	for b.Loop() {
		tr := rbtree.New[string, Book](cmp.Compare[string])
		for _, bk := range books {
			tr.Insert(bk.ISBN, bk)
		}
	}
}

func benchRBTreeInsertVOnly(b *testing.B, n int) {
	books := makeBooks(n)
	less := func(a, b Book) int { return cmp.Compare(a.ISBN, b.ISBN) }
	b.ResetTimer()
	for b.Loop() {
		tr := rbtree.New[Book, struct{}](less)
		for _, bk := range books {
			tr.Insert(bk, struct{}{})
		}
	}
}

func benchRBTreeFindKStored(b *testing.B, n int) {
	books := makeBooks(n)
	tr := rbtree.New[string, Book](cmp.Compare[string])
	for _, bk := range books {
		tr.Insert(bk.ISBN, bk)
	}
	b.ResetTimer()
	for i := range b.N {
		_ = tr.Find(books[i%n].ISBN)
	}
}

func benchRBTreeFindVOnly(b *testing.B, n int) {
	books := makeBooks(n)
	less := func(a, b Book) int { return cmp.Compare(a.ISBN, b.ISBN) }
	tr := rbtree.New[Book, struct{}](less)
	for _, bk := range books {
		tr.Insert(bk, struct{}{})
	}
	probe := Book{}
	b.ResetTimer()
	for i := range b.N {
		probe.ISBN = books[i%n].ISBN
		_ = tr.Find(probe)
	}
}

func BenchmarkRBTreeInsert_KStored(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchRBTreeInsertKStored(b, n) })
	}
}
func BenchmarkRBTreeInsert_VOnly(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchRBTreeInsertVOnly(b, n) })
	}
}
func BenchmarkRBTreeFind_KStored(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchRBTreeFindKStored(b, n) })
	}
}
func BenchmarkRBTreeFind_VOnly(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchRBTreeFindVOnly(b, n) })
	}
}

// ---- swisstable: K-stored vs V-only Insert/Find (V-only sim is pessimistic) ----

func benchSwissInsertKStored(b *testing.B, n int) {
	books := makeBooks(n)
	b.ResetTimer()
	for b.Loop() {
		m := swisstable.NewWithCapacity[string, Book](n)
		for _, bk := range books {
			m.Insert(bk.ISBN, bk)
		}
	}
}

func benchSwissInsertVOnly(b *testing.B, n int) {
	books := makeBooks(n)
	b.ResetTimer()
	for b.Loop() {
		m := swisstable.NewWithCapacity[Book, struct{}](n)
		for _, bk := range books {
			m.Insert(bk, struct{}{})
		}
	}
}

func benchSwissFindKStored(b *testing.B, n int) {
	books := makeBooks(n)
	m := swisstable.NewWithCapacity[string, Book](n)
	for _, bk := range books {
		m.Insert(bk.ISBN, bk)
	}
	b.ResetTimer()
	for i := range b.N {
		_ = m.Find(books[i%n].ISBN)
	}
}

func benchSwissFindVOnly(b *testing.B, n int) {
	books := makeBooks(n)
	m := swisstable.NewWithCapacity[Book, struct{}](n)
	for _, bk := range books {
		m.Insert(bk, struct{}{})
	}
	b.ResetTimer()
	for i := range b.N {
		_ = m.Find(books[i%n])
	}
}

func BenchmarkSwissInsert_KStored(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchSwissInsertKStored(b, n) })
	}
}
func BenchmarkSwissInsert_VOnly(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchSwissInsertVOnly(b, n) })
	}
}
func BenchmarkSwissFind_KStored(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchSwissFindKStored(b, n) })
	}
}
func BenchmarkSwissFind_VOnly(b *testing.B) {
	for _, n := range sizes {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) { benchSwissFindVOnly(b, n) })
	}
}
