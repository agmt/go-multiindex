package multiindex_test

import (
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/agmt/go-multiindex"
	"github.com/agmt/go-multiindex/multiindex_container"
)

type isRbTree interface {
	IsRbTree() (bool, error)
}

func Verify[V comparable](m *multiindex.MultiIndex[V]) (err error) {
	if len(m.MultiIndexBy) == 0 {
		return nil
	}
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()

	allValues := make(map[V]int)

	for i, cont := range m.MultiIndexBy {
		if t, ok := cont.(isRbTree); ok {
			okRb, err := t.IsRbTree()
			if err != nil {
				return err
			}
			if !okRb {
				return fmt.Errorf("index %d is not a valid RbTree", i)
			}
		}
		if i == 0 {
			cont.TraversalValue(func(v V) bool {
				_, ok := allValues[v]
				if ok {
					panic(fmt.Errorf("duplicate at %d: '%+v'", i, v))
				}
				allValues[v] = 1
				return true
			})
		} else {
			if len(allValues) != cont.Size() {
				panic(fmt.Errorf("wrong reported size at %d: %d != %d", i, len(allValues), cont.Size()))
			}
			cnt := 0
			cont.TraversalValue(func(v V) bool {
				_, ok := allValues[v]
				if !ok {
					panic(fmt.Errorf("exists only at %d: '%+v'", i, v))
				}
				cnt += 1
				return true
			})
			if len(allValues) != cont.Size() {
				panic(fmt.Errorf("wrong real size at %d: %d != %d", i, len(allValues), cont.Size()))
			}
		}
	}

	return nil
}

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

type Allable[K, V any] interface {
	All() iter.Seq2[K, V]
}

type Whereable[K, V any] interface {
	Where(K) iter.Seq[V]
}

type Rangeable[K, V any] interface {
	Range(K, K) iter.Seq2[K, V]
}

func testAll[T comparable](t *testing.T, f Allable[T, Book], expectedCnt int) {
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

func testWhere[T comparable](t *testing.T, f Whereable[T, Book], key T, expectedCount int) {
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

	if err := Verify(m); err != nil {
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

	if err := Verify(m); err != nil {
		t.Errorf("%v", err)
	}

	ok := m.Insert(book3)
	if ok {
		t.Errorf("duplicate inserted")
	}

	if err := Verify(m); err != nil {
		t.Errorf("%v", err)
	}

	testAll(t, byISBNOrdered, 3)
	testAll(t, byAuthorOrdered, 3)
	testAll(t, byISBNNonOrdered, 3)
	testAll(t, byAuthorNonOrdered, 3)
	testAll(t, byAuthorName, 3)

	testWhere(t, byISBNOrdered, "9780000003", 1)
	testWhere(t, byAuthorOrdered, "Herbert George Wells", 2)
	testWhere(t, byISBNNonOrdered, "9780000003", 1)
	testWhere(t, byAuthorNonOrdered, "Herbert George Wells", 2)

	ok = m.Insert(book4)
	if !ok {
		t.Errorf("2nd edition is not inserted")
	}

	testAll(t, byISBNOrdered, 4)
	testAll(t, byAuthorOrdered, 4)
	testAll(t, byISBNNonOrdered, 4)
	testAll(t, byAuthorNonOrdered, 4)
	testAll(t, byAuthorName, 4)

	testWhere(t, byISBNOrdered, "9780000003", 1)
	testWhere(t, byAuthorOrdered, "Herbert George Wells", 3)
	testWhere(t, byISBNNonOrdered, "9780000003", 1)
	testWhere(t, byAuthorNonOrdered, "Herbert George Wells", 3)
	testWhere(t, byISBNOrdered, "9780000023", 1)
	testWhere(t, byISBNNonOrdered, "9780000003", 1)
	testWhere(t, byAuthorName, AuthorName{Author: book4.Author, Name: book4.Name}, 2)

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

	if err := Verify(m); err != nil {
		t.Errorf("%v", err)
	}

	m.Reset()

	if m.Size() != 0 {
		t.Errorf("not reseted")
	}

	if err := Verify(m); err != nil {
		t.Errorf("%v", err)
	}
}
