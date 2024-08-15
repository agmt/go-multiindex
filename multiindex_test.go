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

type Rangeable[K, V any] interface {
	All() iter.Seq2[K, V]
}

type RangeableKey[K, V any] interface {
	Where(K) iter.Seq[V]
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

	byName := multiindex_container.NewOrderedUnique(func(b Book) string { return b.Name })
	byAuthor := multiindex_container.NewOrderedNonUnique(func(b Book) string { return b.Author })
	byISBN := multiindex_container.NewNonOrderedUnique(func(b Book) string { return b.ISBN })
	byAuthorNonOrdered := multiindex_container.NewNonOrderedNonUnique(func(b Book) string { return b.Author })

	m.AddIndex(byName, byAuthor, byISBN, byAuthorNonOrdered)

	m.Insert(book1)
	m.Insert(book2)

	bookIt := byName.Find(book1.Name)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthor.Find(book1.Author)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byISBN.Find(book1.ISBN)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthorNonOrdered.Find(book1.Author)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}

	bookIt = byName.Find(book2.Name)
	if bookIt.Value() != book2 {
		t.Errorf("%v != %v", bookIt.Value(), book2)
	}
	bookIt = byAuthor.Find(book2.Author)
	if bookIt.Value() != book2 {
		t.Errorf("%v != %v", bookIt.Value(), book2)
	}
	bookIt = byISBN.Find(book2.ISBN)
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

	bookIt = byName.Find(book3.Name)
	if bookIt.Value() != book3 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthor.Find(book3.Author)
	if bookIt.Value() != book2 && bookIt.Value() != book3 {
		t.Errorf("%v != %v or %v", bookIt.Value(), book2, book3)
	}
	bookIt = byISBN.Find(book3.ISBN)
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

	testRange := func(f Rangeable[string, Book]) {
		count := 0
		for k, v := range f.All() {
			_ = k
			_ = v
			count += 1
		}
		if count != 3 {
			t.Errorf("count: %d != 3", count)
		}
	}
	testRange(byName)
	testRange(byAuthor)
	testRange(byISBN)
	testRange(byAuthorNonOrdered)

	testRangeKey := func(f RangeableKey[string, Book], key string, expectedCount int) {
		count := 0
		for v := range f.Where(key) {
			_ = v
			count += 1
		}
		if count != expectedCount {
			t.Errorf("count: %d != %d", count, expectedCount)
		}
	}
	testRangeKey(byName, "Around the World in Eighty Days", 1)
	testRangeKey(byAuthor, "Herbert George Wells", 2)
	testRangeKey(byISBN, "9780000003", 1)
	testRangeKey(byAuthorNonOrdered, "Herbert George Wells", 2)

	for {
		it := byAuthor.Find(book2.Author)
		if !it.IsValid() {
			break
		}
		m.Erase(it.Value())
	}
	bookIt = byName.Find(book1.Name)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byAuthor.Find(book1.Author)
	if bookIt.Value() != book1 {
		t.Errorf("%v != %v", bookIt.Value(), book1)
	}
	bookIt = byISBN.Find(book1.ISBN)
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
