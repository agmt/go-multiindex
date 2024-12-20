package multiindex_test

import (
	"iter"
	"testing"
	"time"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/multiindex_container"
)

type Book struct {
	Name        string
	Author      string
	ISBN        string
	PublushedAt time.Time
}

type AuthorName struct {
	Author string
	Name   string
}

type Rangeable[K, V any] interface {
	All() iter.Seq2[K, V]
}

type RangeableKey[K, V any] interface {
	Where(K) iter.Seq[V]
}

func testRange[T comparable](t *testing.T, f Rangeable[T, Book], expectedCnt int) {
	count := 0
	for k, v := range f.All() {
		_ = k
		_ = v
		count += 1
	}
	if count != expectedCnt {
		t.Errorf("count: %d != 3", count)
	}
}

func testRangeKey[T comparable](t *testing.T, f RangeableKey[T, Book], key T, expectedCount int) {
	count := 0
	for v := range f.Where(key) {
		_ = v
		count += 1
	}
	if count != expectedCount {
		t.Errorf("count: %d != %d", count, expectedCount)
	}
}

func TestMapOrdOrd(t *testing.T) {
	m := multiindex.New[Book]()
	book1 := Book{
		Name:        "Around the World in Eighty Days",
		Author:      "Jules Verne",
		ISBN:        "9780000001",
		PublushedAt: time.Time{},
	}
	book2 := Book{
		Name:        "The Time Machine",
		Author:      "Herbert George Wells",
		ISBN:        "9780000002",
		PublushedAt: time.Time{},
	}

	book3 := Book{
		Name:        "The Invisible Man",
		Author:      "Herbert George Wells",
		ISBN:        "9780000003",
		PublushedAt: time.Time{},
	}

	book4 := Book{
		Name:        "The Invisible Man",
		Author:      "Herbert George Wells",
		ISBN:        "9780000023", // 2nd Edition
		PublushedAt: time.Time{},
	}

	byISBNOrdered := multiindex_container.NewOrderedUnique(func(b Book) string { return b.ISBN })
	byAuthorOrdered := multiindex_container.NewOrderedNonUnique(func(b Book) string { return b.Author })
	byISBNNonOrdered := multiindex_container.NewNonOrderedUnique(func(b Book) string { return b.ISBN })
	byAuthorNonOrdered := multiindex_container.NewNonOrderedNonUnique(func(b Book) string { return b.Author })
	byAuthorName := multiindex_container.NewNonOrderedNonUnique(func(b Book) AuthorName { return AuthorName{Author: b.Author, Name: b.Name} })

	m.AddIndex(
		byISBNOrdered,
		byAuthorOrdered,
		byISBNNonOrdered,
		byAuthorNonOrdered,
		byAuthorName,
	)

	m.Insert(book1)
	m.Insert(book2)

	bookIt := byISBNOrdered.Find(book1.ISBN)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthorOrdered.Find(book1.Author)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byISBNNonOrdered.Find(book1.ISBN)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthorNonOrdered.Find(book1.Author)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}

	bookIt = byISBNOrdered.Find(book2.ISBN)
	if bookIt.Value() != book2 {
		t.Errorf("%v != %v", bookIt.Value(), book2)
	}
	bookIt = byAuthorOrdered.Find(book2.Author)
	if bookIt.Value() != book2 {
		t.Errorf("%v != %v", bookIt.Value(), book2)
	}
	bookIt = byISBNNonOrdered.Find(book2.ISBN)
	if bookIt.Value() != book2 {
		t.Errorf("%v != %v", bookIt.Value(), book2)
	}
	bookIt = byAuthorNonOrdered.Find(book2.Author)
	if bookIt.Value() != book2 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}

	if err := m.Verify(); err != nil {
		t.Errorf("%v", err)
	}

	m.Insert(book3)

	bookIt = byISBNOrdered.Find(book3.ISBN)
	if bookIt.Value() != book3 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthorOrdered.Find(book3.Author)
	if bookIt.Value() != book2 && bookIt.Value() != book3 {
		t.Errorf("%v != %v or %v", bookIt.Value(), book2, book3)
	}
	bookIt = byISBNNonOrdered.Find(book3.ISBN)
	if bookIt.Value() != book3 {
		t.Errorf("%v != %v", bookIt.Value(), book3)
	}
	bookIt = byAuthorNonOrdered.Find(book3.Author)
	if bookIt.Value() != book2 && bookIt.Value() != book3 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}

	if err := m.Verify(); err != nil {
		t.Errorf("%v", err)
	}

	ok := m.Insert(book3)
	if ok {
		t.Errorf("duplicate inserted")
	}

	if err := m.Verify(); err != nil {
		t.Errorf("%v", err)
	}

	testRange(t, byISBNOrdered, 3)
	testRange(t, byAuthorOrdered, 3)
	testRange(t, byISBNNonOrdered, 3)
	testRange(t, byAuthorNonOrdered, 3)
	testRange(t, byAuthorName, 3)

	testRangeKey(t, byISBNOrdered, "9780000003", 1)
	testRangeKey(t, byAuthorOrdered, "Herbert George Wells", 2)
	testRangeKey(t, byISBNNonOrdered, "9780000003", 1)
	testRangeKey(t, byAuthorNonOrdered, "Herbert George Wells", 2)

	ok = m.Insert(book4)
	if !ok {
		t.Errorf("2nd edition is not inserted")
	}

	testRange(t, byISBNOrdered, 4)
	testRange(t, byAuthorOrdered, 4)
	testRange(t, byISBNNonOrdered, 4)
	testRange(t, byAuthorNonOrdered, 4)
	testRange(t, byAuthorName, 4)

	testRangeKey(t, byISBNOrdered, "9780000003", 1)
	testRangeKey(t, byAuthorOrdered, "Herbert George Wells", 3)
	testRangeKey(t, byISBNNonOrdered, "9780000003", 1)
	testRangeKey(t, byAuthorNonOrdered, "Herbert George Wells", 3)
	testRangeKey(t, byISBNOrdered, "9780000023", 1)
	testRangeKey(t, byISBNNonOrdered, "9780000003", 1)
	testRangeKey(t, byAuthorName, AuthorName{Author: book4.Author, Name: book4.Name}, 2)

	for {
		it := byAuthorOrdered.Find(book2.Author)
		if !it.IsValid() {
			break
		}
		m.Erase(it.Value())
	}
	bookIt = byISBNOrdered.Find(book1.ISBN)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthorOrdered.Find(book1.Author)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byISBNNonOrdered.Find(book1.ISBN)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	if m.Size() != 1 {
		t.Errorf("not removed")
	}

	if err := m.Verify(); err != nil {
		t.Errorf("%v", err)
	}
}
