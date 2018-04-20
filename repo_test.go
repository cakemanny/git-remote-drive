package main

import (
	"errors"
	"io"
	"testing"

	store "github.com/cakemanny/git-remote-drive/store"
)

// mapStore is rooted at .git
type mapStore struct {
	contents map[string]string
	listings map[string][]store.File
}

var fakeStore store.SimpleFileStore = mapStore{
	contents: map[string]string{
		"refs/heads/master": "c5d2d737af4b6203aa37ca2ca13476624d11f4ee\n",
	},
	listings: map[string][]store.File{
		"":           []store.File{{true, "objects"}, {true, "refs"}},
		"refs":       []store.File{{true, "heads"}},
		"refs/heads": []store.File{{false, "master"}},
		"objects":    []store.File{{true, "c5"}},
		"objects/c5": []store.File{
			{false, "d2d737af4b6203aa37ca2ca13476624d11f4ee"},
		},
	},
}

var basedStore store.SimpleFileStore = mapStore{
	contents: map[string]string{
		".git/refs/heads/master": "c5d2d737af4b6203aa37ca2ca13476624d11f4ee\n",
	},
	listings: map[string][]store.File{
		"":                []store.File{{true, ".git"}},
		".git":            []store.File{{true, "objects"}, {true, "refs"}},
		".git/refs":       []store.File{{true, "heads"}},
		".git/refs/heads": []store.File{{false, "master"}},
		".git/objects":    []store.File{{true, "c5"}},
		".git/objects/c5": []store.File{
			{false, "d2d737af4b6203aa37ca2ca13476624d11f4ee"},
		},
	},
}

func (s mapStore) Read(name string, contents io.Writer) error {
	m := s.contents
	v, ok := m[name]
	if !ok {
		return store.ErrNotFound
	}
	contents.Write([]byte(v))
	return nil
}
func (s mapStore) List(path string) ([]store.File, error) {
	m := s.listings
	v, ok := m[path]
	if !ok {
		return nil, store.ErrNotFound
	}
	return v, nil
}
func (mapStore) Create(path string, contents io.Reader) error {
	return errors.New("not implemented")
}
func (mapStore) Update(path string, contents io.Reader) error {
	return errors.New("not implemented")
}
func (mapStore) Delete(path string) error {
	return errors.New("not implemented")
}

func TestReadRef(t *testing.T) {

	stores := []struct {
		storeName string
		base      string
		store     store.SimpleFileStore
	}{
		{"fakeStore", "", fakeStore},
		{"basedStore", ".git", basedStore},
	}

	for _, s := range stores {

		m := storeManager{s.base, s.store}

		sha1, err := m.ReadRef("refs/heads/master")
		if err != nil {
			t.Errorf("%s: ReadRef: unexpected error: %v", s.storeName, err)
		}

		expected := "c5d2d737af4b6203aa37ca2ca13476624d11f4ee"
		if sha1 != expected {
			t.Error(s.storeName+":", "expected:", expected, "actual:", sha1)
		}
	}
}
