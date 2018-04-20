package store

import (
	"errors"
	"io"
	paths "path"
)

var (
	// ErrNotFound is returned when a file cannot be found
	ErrNotFound = errors.New("not found")
)

// File represents a filesystem entry
type File struct {
	IsFolder bool
	Name     string
}

type ReadOnlyStore interface {
	Read(path string, contents io.Writer) error
	List(path string) ([]File, error)
}

// SimpleFileStore is a simplied interface over a file store
type SimpleFileStore interface {
	Create(path string, contents io.Reader) error
	Read(path string, contents io.Writer) error
	Update(path string, contents io.Reader) error
	Delete(path string) error
	List(path string) ([]File, error)
}

// define this to allow us to unit test the recursive ID getter
type idGetter interface {
	GetID(name string, parentID string) (string, error)
}

// GetIDRecursive navigates up the directory tree
func GetIDRecursive(client idGetter, path string) (string, error) {
	parentPath, name := paths.Dir(path), paths.Base(path)
	if parentPath == "." {
		// No more parent IDs to get
		return client.GetID(name, "")
	}
	// Not root level. Find in parent
	parentID, err := GetIDRecursive(client, parentPath)
	if err != nil {
		return "", err
	}
	return client.GetID(name, parentID)
}

// TODO: implement caching around ID getter
