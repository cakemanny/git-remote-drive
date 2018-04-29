package main

import (
	"io"
	"strings"
	"testing"

	errors "github.com/cakemanny/git-remote-drive/errors"
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
		return errors.ErrNotFound{name}
	}
	contents.Write([]byte(v))
	return nil
}
func (s mapStore) List(path string) ([]store.File, error) {
	m := s.listings
	v, ok := m[path]
	if !ok {
		return nil, errors.ErrNotFound{path}
	}
	return v, nil
}
func (mapStore) Create(path string, contents io.Reader) error {
	return errors.NotImplemented()
}
func (mapStore) Update(path string, contents io.Reader) error {
	return errors.NotImplemented()
}
func (mapStore) Delete(path string) error {
	return errors.NotImplemented()
}
func (s mapStore) TestPath(path string) (bool, error) {
	m := s.listings
	_, ok := m[path]
	return ok, nil
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

func TestReadCommit(t *testing.T) {
	rdr := strings.NewReader(
		"tree 07b2986536979a8e6b6028c6a670012b4b4ac262\n" +
			"parent 7879dfcfd2db5c052284d7077441e9500672a702\n" +
			"parent 35a3be730435891d106bd4a7eefba3183ab14d54\n" +
			"author cakemanny <goldingd89@gmail.com> 1524238149 +0100\n" +
			"committer cakemanny <goldingd89@gmail.com> 1524238149 +0100\n" +
			"\n" +
			"Merge branch 'branch1'\n",
	)

	expected := Commit{
		Tree: "07b2986536979a8e6b6028c6a670012b4b4ac262",
		Parents: []string{
			"7879dfcfd2db5c052284d7077441e9500672a702",
			"35a3be730435891d106bd4a7eefba3183ab14d54",
		},
	}

	actual, err := ReadCommit(rdr)
	if err != nil {
		t.Error("unexpected error:", err)
	}

	// can't directly compare
	if expected.Tree != actual.Tree ||
		len(expected.Parents) != len(actual.Parents) ||
		expected.Parents[0] != actual.Parents[0] ||
		expected.Parents[1] != actual.Parents[1] {
		//
		t.Error("expected:", expected, "actual:", actual)
	}
}

func TestReadTree(t *testing.T) {
	rdr := strings.NewReader(
		"100644 blob 7311bd3ca3f61d3731a390d88422977b8d23a016\tREADME\n" +
			"040000 tree 562d81834eec9f1701b7ee35ea50767edb2c4e8a\tsomedir\n",
	)

	expected := Tree{
		{Type: BLOB, Ref: "7311bd3ca3f61d3731a390d88422977b8d23a016"},
		{Type: TREE, Ref: "562d81834eec9f1701b7ee35ea50767edb2c4e8a"},
	}

	actual, err := ReadTree(rdr)
	if err != nil {
		t.Error("unexpected error:", err)
	}

	if len(actual) != 2 || expected[0] != actual[0] || expected[1] != actual[1] {
		t.Error("expected:", expected, "actual:", actual)
	}
}

//
