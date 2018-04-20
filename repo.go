package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"unicode"

	store "github.com/cakemanny/git-remote-drive/store"
)

// Manager is an abstraction over some of the simplest operations you might
// want to perform against a git repository
type Manager interface {
	ListRefs() ([]Ref, error)
	ReadRef(name string) (string, error)
	WriteRef(ref Ref) error
	ReadObject(sha string, contents io.Writer) error
	ReadRaw(sha string, contents io.Writer) error
	WriteRaw(sha string, contents io.Reader) error
}

type RefLister interface {
	ListRefs() ([]Ref, error)
}
type RefReader interface {
	ReadRef(name string) (string, error)
}

// prove at compiletime that RefLister is a subinterface on Manager
var manager0 Manager
var refLister0 RefLister = manager0

// Ref represents a git references
type Ref struct {
	// c5d2d737af4b6203aa37ca2ca13476624d11f4ee
	Value string
	// refs/heads/master
	Name string
}

type storeManager struct {
	// relative path from the root of the store
	// e.g. ".git" for a local repo or "user/reponame.git" for a remote say
	basePath string
	store    store.SimpleFileStore
}

func (m storeManager) ListRefs() ([]Ref, error) {
	var results []Ref
	pathToRefs := path.Join(m.basePath, "refs")

	onFile := func(p string) error {
		if options.verbosity >= 1 {
			log.Printf("reading file \"%s\"", p)
		}
		var sha1val strings.Builder
		if err := m.store.Read(p, &sha1val); err != nil {
			return fmt.Errorf("error reading \"%s\" in remote store: %v", p, err)
		}
		name := strings.TrimPrefix(strings.TrimPrefix(p, m.basePath), "/")
		results = append(results, Ref{
			strings.TrimRightFunc(sha1val.String(), unicode.IsSpace),
			name,
		})
		return nil
	}

	var walk func(string) error
	walk = func(baseDir string) error {
		if options.verbosity >= 1 {
			log.Printf("walking \"%s\"", baseDir)
		}
		list, err := m.store.List(baseDir)
		if err == store.ErrNotFound {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unable to list \"%s\" directory: %v", baseDir, err)
		}
		for _, entry := range list {
			p := path.Join(baseDir, entry.Name)
			if entry.IsFolder {
				if err := walk(p); err != nil {
					return err
				}
			} else {
				if err := onFile(p); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(pathToRefs); err != nil {
		return nil, fmt.Errorf("walking refs dir: %v", err)
	}
	return results, nil
}

func (m storeManager) objectPath(sha string) (string, error) {
	if len(sha) != 40 {
		return "", fmt.Errorf("invalid sha: \"%s\"", sha)
	}
	//    c5d2d737af4b6203aa37ca2ca13476624d11f4ee
	// -> objects/c5/d2d737af4b6203aa37ca2ca13476624d11f4ee
	subDir := sha[:2]
	fileName := sha[2:]
	return path.Join(m.basePath, "objects", subDir, fileName), nil
}

func (m storeManager) ReadRaw(sha string, contents io.Writer) error {
	fullPath, err := m.objectPath(sha)
	if err != nil {
		return err
	}
	return m.store.Read(fullPath, contents)
}

func (m storeManager) WriteRaw(sha string, contents io.Reader) error {
	fullPath, err := m.objectPath(sha)
	if err != nil {
		return err
	}
	// Test path? If not exists create?
	// We surely don't need to "update" any objects
	return m.store.Update(fullPath, contents)
}

func (storeManager) ReadObject(sha string, contents io.Writer) error {
	return errors.New("not implemented")
}

func (m storeManager) ReadRef(name string) (string, error) {
	fullPath := path.Join(m.basePath, name)
	var sb strings.Builder
	if err := m.store.Read(fullPath, &sb); err != nil {
		return "", err
	}
	// the ref file usually ends in a \n so trim that
	return strings.TrimRight(sb.String(), "\n"), nil
}

func (storeManager) WriteRef(ref Ref) error {
	return errors.New("not implemented")
}
