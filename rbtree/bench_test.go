package rbtree

import (
	"cmp"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// Benchmarks for rbtree.Tree. The builtin map comparisons aren't apples-to-
// apples (map is unordered, rbtree is ordered), but they give a sense of the
// cost of ordered ops vs unordered for the same workload.

var sizes = []int{64, 1_000, 100_000}

func intKeys(n int) []int {
	rng := rand.New(rand.NewPCG(0xdead, 0xbeef))
	xs := make([]int, n)
	for i := range xs {
		xs[i] = int(rng.Uint64())
	}
	return xs
}

// ---- Insert ----

func BenchmarkInsert_RbTree(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				tr := New[int, int](cmp.Compare[int])
				for j, k := range keys {
					tr.Insert(k, j)
				}
			}
		})
	}
}

func BenchmarkInsert_BuiltinMap(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				m := make(map[int]int, n)
				for j, k := range keys {
					m[k] = j
				}
			}
		})
	}
}

// ---- Find: hits ----

func BenchmarkFindHit_RbTree(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		tr := New[int, int](cmp.Compare[int])
		for j, k := range keys {
			tr.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for i := range b.N {
				it := tr.Find(keys[i%n])
				sink ^= it.Value()
			}
			_ = sink
		})
	}
}

func BenchmarkFindHit_BuiltinMap(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		m := make(map[int]int, n)
		for j, k := range keys {
			m[k] = j
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for i := range b.N {
				sink ^= m[keys[i%n]]
			}
			_ = sink
		})
	}
}

// ---- Find: misses ----

func BenchmarkFindMiss_RbTree(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		miss := intKeys(n)
		for i := range miss {
			miss[i] ^= 0x1
		}
		tr := New[int, int](cmp.Compare[int])
		for j, k := range keys {
			tr.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if tr.Find(miss[i%n]).IsValid() {
					hits++
				}
			}
			_ = hits
		})
	}
}

// ---- Ordered iteration (the real reason to use a tree) ----

func BenchmarkIterate_RbTree(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		tr := New[int, int](cmp.Compare[int])
		for j, k := range keys {
			tr.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				for it := tr.Begin(); it.IsValid(); it = it.Next() {
					sink ^= it.Value()
				}
			}
			_ = sink
		})
	}
}

func BenchmarkIterate_RbTree_RangeFunc(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		tr := New[int, int](cmp.Compare[int])
		for j, k := range keys {
			tr.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				for _, v := range tr.All() {
					sink ^= v
				}
			}
			_ = sink
		})
	}
}

// Equivalent for builtin map: collect keys, sort, then iterate in order.
// This is what people who want sorted iteration with a hash map have to do.
func BenchmarkIterate_BuiltinMapSorted(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		m := make(map[int]int, n)
		for j, k := range keys {
			m[k] = j
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				ks := make([]int, 0, len(m))
				for k := range m {
					ks = append(ks, k)
				}
				slices.Sort(ks)
				for _, k := range ks {
					sink ^= m[k]
				}
			}
			_ = sink
		})
	}
}

// ---- Range query ----

func BenchmarkRangeQuery_RbTree(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		tr := New[int, int](cmp.Compare[int])
		for j, k := range keys {
			tr.Insert(k, j)
		}
		sorted := slices.Clone(keys)
		slices.Sort(sorted)
		// Query the middle 10% of the keyspace.
		lo := sorted[n/2-n/20]
		hi := sorted[n/2+n/20]
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				for it := tr.LowerBound(lo); it.IsValid() && it.Key() < hi; it = it.Next() {
					sink ^= it.Value()
				}
			}
			_ = sink
		})
	}
}

// ---- Mixed churn: insert/erase alternation ----

func BenchmarkChurn_RbTree(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			tr := New[int, int](cmp.Compare[int])
			for j, k := range keys {
				tr.Insert(k, j)
			}
			b.ResetTimer()
			for i := range b.N {
				k := keys[i%n]
				if i&1 == 0 {
					if it := tr.Find(k); it.IsValid() {
						tr.Erase(it)
					}
				} else {
					tr.Insert(k, i)
				}
			}
		})
	}
}
