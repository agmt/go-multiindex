package multiindex_test

import (
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

	byName := multiindex_container.NewOrderedUnique(m, func(b Book) string { return b.Name })
	byAuthor := multiindex_container.NewOrderedNonUnique(m, func(b Book) string { return b.Author })
	byISBN := multiindex_container.NewNonOrderedUnique(m, func(b Book) string { return b.ISBN })
	byAuthorNonOrdered := multiindex_container.NewNonOrderedNonUnique(m, func(b Book) string { return b.Author })

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

	for {
		it := byAuthor.Find(book2.Author)
		if !it.IsValid() {
			break
		}
		m.EraseValue(it.Value())
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
