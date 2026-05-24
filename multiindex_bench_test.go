package multiindex_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/multiindex_container"
)

// End-to-end benchmarks against the same 5-index shape used by
// TestMapOrdOrd: byISBN ordered_unique, byAuthor ordered_nonunique,
// byISBN nonordered_unique, byAuthor nonordered_nonunique, and a
// composite-key (Author, Name) nonordered_nonunique.

var benchSizes = []int{1_000, 100_000}

func makeBenchBooks(n int) []Book {
	rng := rand.New(rand.NewPCG(0x1234, 0x5678))
	books := make([]Book, n)
	for i := range books {
		books[i] = Book{
			ISBN:        fmt.Sprintf("978-%013d", i),                 // unique
			Author:      fmt.Sprintf("Author-%04d", rng.IntN(n/10+1)), // ~10 books / author on average
			Name:        fmt.Sprintf("Title-%08d", i),                 // unique-ish
			PublushedAt: time.Time{},
		}
	}
	return books
}

// indexSet bundles the five indexes plus the top-level MultiIndex so we can
// reach individual containers for Find benchmarks while still routing
// Insert/Erase through the orchestration layer.
type indexSet struct {
	mi *multiindex.MultiIndex[Book]

	byISBNOrdered    *multiindex_container.MultiIndexByOrderedUnique[string, Book]
	byAuthorOrdered  *multiindex_container.MultiIndexByOrderedNonUnique[string, Book]
	byISBNNonOrd     *multiindex_container.MultiIndexByNonOrderedUnique[string, Book]
	byAuthorNonOrd   *multiindex_container.MultiIndexByNonOrderedNonUnique[string, Book]
	byAuthorName     *multiindex_container.MultiIndexByNonOrderedNonUnique[AuthorName, Book]
}

func newIndexSet(tb testing.TB) *indexSet {
	tb.Helper()
	s := &indexSet{mi: multiindex.New[Book]()}
	s.byISBNOrdered = multiindex_container.NewOrderedUnique(func(b Book) string { return b.ISBN })
	s.byAuthorOrdered = multiindex_container.NewOrderedNonUnique(func(b Book) string { return b.Author })
	s.byISBNNonOrd = multiindex_container.NewNonOrderedUnique(func(b Book) string { return b.ISBN })
	s.byAuthorNonOrd = multiindex_container.NewNonOrderedNonUnique(func(b Book) string { return b.Author })
	s.byAuthorName = multiindex_container.NewNonOrderedNonUnique(func(b Book) AuthorName {
		return AuthorName{Author: b.Author, Name: b.Name}
	})
	if err := s.mi.AddIndex(s.byISBNOrdered, s.byAuthorOrdered, s.byISBNNonOrd, s.byAuthorNonOrd, s.byAuthorName); err != nil {
		tb.Fatalf("AddIndex: %v", err)
	}
	return s
}

func (s *indexSet) populate(books []Book) {
	for _, b := range books {
		s.mi.Insert(b)
	}
}

// ---- Insert (write-heavy) ----

func BenchmarkMultiIndex_Insert(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				s := newIndexSet(b)
				for _, bk := range books {
					s.mi.Insert(bk)
				}
			}
		})
	}
}

// ---- Find via each index ----

func BenchmarkMultiIndex_FindByISBNOrdered(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		s := newIndexSet(b)
		s.populate(books)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if s.byISBNOrdered.Find(books[i%n].ISBN).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

func BenchmarkMultiIndex_FindByAuthorOrdered(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		s := newIndexSet(b)
		s.populate(books)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if s.byAuthorOrdered.Find(books[i%n].Author).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

func BenchmarkMultiIndex_FindByISBNNonOrdered(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		s := newIndexSet(b)
		s.populate(books)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if s.byISBNNonOrd.Find(books[i%n].ISBN).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

func BenchmarkMultiIndex_FindByAuthorNonOrdered(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		s := newIndexSet(b)
		s.populate(books)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if s.byAuthorNonOrd.Find(books[i%n].Author).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

func BenchmarkMultiIndex_FindByAuthorName(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		s := newIndexSet(b)
		s.populate(books)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				probe := AuthorName{Author: books[i%n].Author, Name: books[i%n].Name}
				if s.byAuthorName.Find(probe).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

// ---- Erase ----

// Erase repopulates inside the timed loop because the workload destroys the
// index; we time pairs of (populate, erase-all) so the per-op cost reflects
// the full propagation overhead.
func BenchmarkMultiIndex_Erase(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBenchBooks(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				s := newIndexSet(b)
				s.populate(books)
				for _, bk := range books {
					s.mi.Erase(bk)
				}
			}
		})
	}
}
