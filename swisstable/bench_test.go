package swisstable

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// Benchmark workloads compare swisstable.Map against Go's builtin map.
// Each benchmark runs at multiple sizes via b.Run subtests.

var sizes = []int{64, 1_000, 100_000, 1_000_000}

func intKeys(n int) []int {
	rng := rand.New(rand.NewPCG(0xdead, 0xbeef))
	xs := make([]int, n)
	for i := range xs {
		xs[i] = int(rng.Uint64())
	}
	return xs
}

func stringKeys(n int) []string {
	rng := rand.New(rand.NewPCG(0xdead, 0xbeef))
	xs := make([]string, n)
	for i := range xs {
		xs[i] = fmt.Sprintf("key-%016x", rng.Uint64())
	}
	return xs
}

// ---- Insert: pre-sized ----

func BenchmarkInsert_Int_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				m := NewWithCapacity[int, int](n)
				for j, k := range keys {
					m.Insert(k, j)
				}
			}
		})
	}
}

func BenchmarkInsert_Int_Builtin(b *testing.B) {
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

func BenchmarkInsert_String_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := stringKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				m := NewWithCapacity[string, int](n)
				for j, k := range keys {
					m.Insert(k, j)
				}
			}
		})
	}
}

func BenchmarkInsert_String_Builtin(b *testing.B) {
	for _, n := range sizes {
		keys := stringKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for b.Loop() {
				m := make(map[string]int, n)
				for j, k := range keys {
					m[k] = j
				}
			}
		})
	}
}

// ---- Find: hits ----

func BenchmarkFindHit_Int_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		m := NewWithCapacity[int, int](n)
		for j, k := range keys {
			m.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for i := range b.N {
				sink ^= m.Find(keys[i%n]).Value()
			}
			_ = sink
		})
	}
}

func BenchmarkFindHit_Int_Builtin(b *testing.B) {
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

func BenchmarkFindHit_String_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := stringKeys(n)
		m := NewWithCapacity[string, int](n)
		for j, k := range keys {
			m.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for i := range b.N {
				sink ^= m.Find(keys[i%n]).Value()
			}
			_ = sink
		})
	}
}

func BenchmarkFindHit_String_Builtin(b *testing.B) {
	for _, n := range sizes {
		keys := stringKeys(n)
		m := make(map[string]int, n)
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

func BenchmarkFindMiss_Int_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		miss := intKeys(n)
		for i := range miss {
			miss[i] ^= 0x1 // flip a bit; near-certain miss
		}
		m := NewWithCapacity[int, int](n)
		for j, k := range keys {
			m.Insert(k, j)
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

func BenchmarkFindMiss_Int_Builtin(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		miss := intKeys(n)
		for i := range miss {
			miss[i] ^= 0x1
		}
		m := make(map[int]int, n)
		for j, k := range keys {
			m[k] = j
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var hits int
			for i := range b.N {
				if _, ok := m[miss[i%n]]; ok {
					hits++
				}
			}
			_ = hits
		})
	}
}

// ---- Iterate ----

func BenchmarkIterate_Int_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		m := NewWithCapacity[int, int](n)
		for j, k := range keys {
			m.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				for it := m.Begin(); it.IsValid(); it = it.Next() {
					sink ^= it.Value()
				}
			}
			_ = sink
		})
	}
}

func BenchmarkIterate_Int_SwissRangeFunc(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		m := NewWithCapacity[int, int](n)
		for j, k := range keys {
			m.Insert(k, j)
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				for _, v := range m.All() {
					sink ^= v
				}
			}
			_ = sink
		})
	}
}

func BenchmarkIterate_Int_Builtin(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		m := make(map[int]int, n)
		for j, k := range keys {
			m[k] = j
		}
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			var sink int
			for b.Loop() {
				for _, v := range m {
					sink ^= v
				}
			}
			_ = sink
		})
	}
}

// ---- Mixed churn: 50/50 insert/erase over a hot set ----

func BenchmarkChurn_Int_Swiss(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			m := NewWithCapacity[int, int](n)
			for j, k := range keys {
				m.Insert(k, j)
			}
			b.ResetTimer()
			for i := range b.N {
				k := keys[i%n]
				if i&1 == 0 {
					m.Erase(k)
				} else {
					m.Insert(k, i)
				}
			}
		})
	}
}

func BenchmarkChurn_Int_Builtin(b *testing.B) {
	for _, n := range sizes {
		keys := intKeys(n)
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			m := make(map[int]int, n)
			for j, k := range keys {
				m[k] = j
			}
			b.ResetTimer()
			for i := range b.N {
				k := keys[i%n]
				if i&1 == 0 {
					delete(m, k)
				} else {
					m[k] = i
				}
			}
		})
	}
}
