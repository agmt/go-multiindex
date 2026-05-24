package swissmap

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/agmt/go-multiindex/swisstable"
)

// Head-to-head comparison: swissmap.Map[V, K] (key extracted via keyOf,
// stored alongside V) vs swisstable.Map[K, V] (caller passes K explicitly).
// Identical hot path mechanically — confirms the rewrite achieves parity.

var benchSizes = []int{1_000, 100_000}

type bookSlot struct {
	ISBN, Name, Author string
}

func bookSlotISBN(b bookSlot) string { return b.ISBN }

func makeBookSlots(n int) []bookSlot {
	rng := rand.New(rand.NewPCG(0xfeed, 0xface))
	xs := make([]bookSlot, n)
	for i := range xs {
		xs[i] = bookSlot{
			ISBN:   fmt.Sprintf("978-%013d", rng.Uint64()),
			Name:   "name",
			Author: "author",
		}
	}
	return xs
}

func makeKeysAndBooks(n int) ([]string, []bookSlot) {
	books := makeBookSlots(n)
	keys := make([]string, n)
	for i, b := range books {
		keys[i] = b.ISBN
	}
	return keys, books
}

// ---- Insert ----

func BenchmarkInsert_SwissMap(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBookSlots(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				m := NewWithCapacity[bookSlot, string](bookSlotISBN, n)
				for _, bk := range books {
					m.Insert(bk)
				}
			}
		})
	}
}

func BenchmarkInsert_SwissTable(b *testing.B) {
	for _, n := range benchSizes {
		keys, books := makeKeysAndBooks(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				m := swisstable.NewWithCapacity[string, bookSlot](n)
				for i, bk := range books {
					m.Insert(keys[i], bk)
				}
			}
		})
	}
}

// ---- FindHit (the target metric) ----

func BenchmarkFindHit_SwissMap(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBookSlots(n)
		m := NewWithCapacity[bookSlot, string](bookSlotISBN, n)
		for _, bk := range books {
			m.Insert(bk)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if m.Find(books[i%n].ISBN).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

func BenchmarkFindHit_SwissTable(b *testing.B) {
	for _, n := range benchSizes {
		keys, books := makeKeysAndBooks(n)
		m := swisstable.NewWithCapacity[string, bookSlot](n)
		for i, bk := range books {
			m.Insert(keys[i], bk)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if m.Find(keys[i%n]).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

// ---- FindMiss ----

func BenchmarkFindMiss_SwissMap(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBookSlots(n)
		miss := make([]string, n)
		for i := range miss {
			miss[i] = fmt.Sprintf("MISS-%d", i)
		}
		m := NewWithCapacity[bookSlot, string](bookSlotISBN, n)
		for _, bk := range books {
			m.Insert(bk)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if m.Find(miss[i%n]).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

func BenchmarkFindMiss_SwissTable(b *testing.B) {
	for _, n := range benchSizes {
		keys, books := makeKeysAndBooks(n)
		miss := make([]string, n)
		for i := range miss {
			miss[i] = fmt.Sprintf("MISS-%d", i)
		}
		m := swisstable.NewWithCapacity[string, bookSlot](n)
		for i, bk := range books {
			m.Insert(keys[i], bk)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if m.Find(miss[i%n]).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

// ---- Iterate ----

func BenchmarkIterate_SwissMap(b *testing.B) {
	for _, n := range benchSizes {
		books := makeBookSlots(n)
		m := NewWithCapacity[bookSlot, string](bookSlotISBN, n)
		for _, bk := range books {
			m.Insert(bk)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var seen int
			for b.Loop() {
				for v := range m.All() {
					_ = v
					seen++
				}
			}
			_ = seen
		})
	}
}

func BenchmarkIterate_SwissTable(b *testing.B) {
	for _, n := range benchSizes {
		keys, books := makeKeysAndBooks(n)
		m := swisstable.NewWithCapacity[string, bookSlot](n)
		for i, bk := range books {
			m.Insert(keys[i], bk)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var seen int
			for b.Loop() {
				for _, v := range m.All() {
					_ = v
					seen++
				}
			}
			_ = seen
		})
	}
}
