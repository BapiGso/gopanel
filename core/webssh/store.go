package webssh

import (
	"errors"
	"sync"
)

type TermRef struct {
	term   *Term
	refcnt int
}

type TermStore struct {
	sync.Mutex
	terms map[string]*TermRef
}

var termStore = &TermStore{
	terms: map[string]*TermRef{},
}

func (store *TermStore) All() []*Term {
	store.Lock()
	defer store.Unlock()

	terms := []*Term{}
	for _, t := range store.terms {
		terms = append(terms, t.term)
	}

	return terms
}

func (store *TermStore) New(o TermOption) (*Term, error) {
	if o.Port == 0 {
		o.Port = 22
	}

	l := &TermLink{Host: o.Host, Port: o.Port}

	err := l.Dial(o.Username, o.Password)
	if err != nil {
		return nil, err
	}

	if o.Rows == 0 || o.Cols == 0 {
		o.Rows = 40
		o.Cols = 80
	}

	term, err := l.NewTerm(o.Rows, o.Cols)
	if err != nil {
		l.Close()
		return nil, err
	}

	store.Lock()
	defer store.Unlock()

	store.terms[term.Id] = &TermRef{
		term:   term,
		refcnt: 1, // initial refcnt is 1, call put to release.
	}

	return term, nil
}

// Lookup do not increment refcnt!
func (store *TermStore) Lookup(id string) (*Term, error) {
	store.Lock()
	defer store.Unlock()

	r, okay := store.terms[id]
	if !okay {
		return nil, errors.New("term " + id + " not exist")
	}

	return r.term, nil
}

func (store *TermStore) Get(id string) (*Term, error) {
	store.Lock()
	defer store.Unlock()

	r, okay := store.terms[id]
	if !okay {
		return nil, errors.New("term " + id + " not exist")
	}

	r.refcnt += 1

	return r.term, nil
}

func (store *TermStore) Put(id string) {
	store.Lock()
	defer store.Unlock()

	r, okay := store.terms[id]
	if !okay {
		return
	}

	r.refcnt -= 1

	if r.refcnt == 0 {
		delete(store.terms, id)
		r.term.Close()
	}
}

func (store *TermStore) Do(id string, f func(term *Term) error) error {
	store.Lock()
	defer store.Unlock()

	r, ok := store.terms[id]
	if !ok {
		return errors.New("term " + id + " not exist")
	}

	return f(r.term)
}
