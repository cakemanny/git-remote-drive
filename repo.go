package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"unicode"

	errors "github.com/cakemanny/git-remote-drive/errors"
	store "github.com/cakemanny/git-remote-drive/store"
)

// Manager is an abstraction over some of the simplest operations you might
// want to perform against a git repository
type Manager interface {

	// ListRefs lists all the references in a repository. You can assume it
	// does this by scanning the refs directory of the repo.
	ListRefs() ([]Ref, error)

	// ReadRef reads the sha1 stored in a reference. For example, to read the
	// objectname (sha1) of the head commit of the master branch, you might do
	// this:
	//	sha1value, err := manager.ReadRef("refs/heads/master")
	//	if err != nil {
	//		// handle error
	//	}
	//	fmt.Println(sha1value)
	ReadRef(name string) (string, error)

	// WriteRef is the complement of ReadRef. It writes ref.Value to the path
	// ref.Name relative to the repository root.
	WriteRef(ref Ref) error

	// ReadObject reads an object, including decompressing and pretty printing
	// the output. Think git cat-file
	ReadObject(sha string, contents io.Writer) error

	// ReadRaw reads a git repo object without decompressing. Useful for
	// quickly transfering an object without needing to understand it
	ReadRaw(sha string, contents io.Writer) error

	// WriteRaw is the complement to ReadRaw. It writes the contents of an
	// object directly to the git repository assuming the stream is already
	// compressed
	WriteRaw(sha string, contents io.Reader) error
}

// RefLister is a subinterface of Manager only requiring the ListRefs method to
// be implemented.
type RefLister interface {
	ListRefs() ([]Ref, error)
}
type RefReader interface {
	ReadRef(name string) (string, error)
}

// prove at compile-time that RefLister is a subinterface on Manager
var manager0 Manager
var refLister0 RefLister = manager0

// Ref represents a git references
type Ref struct {
	// c5d2d737af4b6203aa37ca2ca13476624d11f4ee
	Value string
	// refs/heads/master
	Name string
}

// Commit represents a git commit object
type Commit struct {

	// Tree is the reference to the tree object
	Tree string

	// Parents is a slice of references to the parent commits
	Parents []string
}

type ObjectType uint

const (
	BLOB ObjectType = iota
	TREE
	COMMIT
	TAG
)

// Tree represents a git tree object
type Tree []struct {
	Type ObjectType
	Ref  string
}

// storeManager is an implementation of the git repo Manager over a
// SimpleFileStore
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
		if _, isNotFound := err.(errors.ErrNotFound); isNotFound {
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

func sha1Bytes(b []byte) (string, error) {
	rdr := bytes.NewReader(b)
	zlibReader, err := zlib.NewReader(rdr)
	if err != nil {
		return "", fmt.Errorf("inflating stream: %v", err)
	}
	defer zlibReader.Close()
	// decompress
	hasher := sha1.New()
	io.Copy(hasher, zlibReader)
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// verifyObject checks that an object is present in the store and the sha1
// of the data matches.
func (m storeManager) verifyObject(sha string) error {
	log.Println("verifying object", sha)
	var buf bytes.Buffer
	if err := m.ReadRaw(sha, &buf); err != nil {
		return err
	}
	actualSha, err := sha1Bytes(buf.Bytes())
	if err != nil {
		return err
	}
	if sha != actualSha {
		return fmt.Errorf("invalid object %s: sha1 of content is %s, "+
			"compressed size is %d bytes", sha, actualSha, len(buf.Bytes()))
	}
	return nil
}

func (m storeManager) WriteRaw(sha string, contents io.Reader) error {
	fullPath, err := m.objectPath(sha)
	if err != nil {
		return err
	}
	// Test path? If not exists create?
	exists, err := m.store.TestPath(fullPath)
	if err != nil {
		return err
	}
	if exists {
		// We surely don't need to "update" any objects
		// so just verify it's ok
		err := m.verifyObject(sha)
		_, invalid := err.(errors.ErrInvalidObject)
		if invalid {
			if deleteErr := m.store.Delete(fullPath); deleteErr != nil {
				return fmt.Errorf(
					"object %s contains invalid data, but cannot be deleted: %v",
					sha,
					deleteErr,
				)
			}
			return m.store.Create(fullPath, contents)
		}
		return err
	}
	return m.store.Create(fullPath, contents)
}

func (storeManager) ReadObject(sha string, contents io.Writer) error {
	return errors.NotImplemented()
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

func (m storeManager) WriteRef(ref Ref) error {
	fullPath := path.Join(m.basePath, ref.Name)
	exists, err := m.store.TestPath(fullPath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(ref.Value)
	buf.WriteByte('\n')
	writeMethod := m.store.Create
	if exists {
		writeMethod = m.store.Update
	}
	if err = writeMethod(fullPath, &buf); err != nil {
		return fmt.Errorf("updating ref %s: %v", ref.Name, err)
	}
	return nil
}

func GetCommit(m Manager, ref string) (Commit, error) {
	if len(ref) != 40 {
		panic(ref)
	}
	var sb strings.Builder
	if err := m.ReadObject(ref, &sb); err != nil {
		return Commit{}, fmt.Errorf("reading object %s: %v", ref, err)
	}
	rdr := strings.NewReader(sb.String())
	return ReadCommit(rdr)
}

func GetTree(m Manager, ref string) (Tree, error) {
	if len(ref) != 40 {
		panic(ref)
	}
	var sb strings.Builder
	if err := m.ReadObject(ref, &sb); err != nil {
		return nil, fmt.Errorf("reading object %s: %v", ref, err)
	}
	rdr := strings.NewReader(sb.String())
	return ReadTree(rdr)
}

// ReadCommit reads commit information from a io.Reader which you can pretend
// supplies the same content as the output of
//	git cat-file -p <ref>
func ReadCommit(rdr io.Reader) (Commit, error) {
	result := Commit{}
	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) == 0 {
			// Remainder will just be commit message
			// consume stream
			for scanner.Scan() {
			}
			break
		}
		if len(fields) == 1 {
			log.Fatalln("ReadCommit: unexpected number of fields in commit line:", line)
		}
		switch fields[0] {
		case "tree":
			result.Tree = fields[1]
		case "parent":
			result.Parents = append(result.Parents, fields[1])
		case "author":
		case "committer":
		default:
			log.Fatalln("ReadCommit: unexpected commit field:", fields[0], fields[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("reading commit object: %v", err)
	}
	return result, nil
}

func ReadTree(rdr io.Reader) (Tree, error) {
	result := Tree{}
	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		// <perms> <type> <sha1>	<filename>
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			log.Fatalln("ReadTree: unexpected number of fields in tree line:", line)
		}
		type Item struct {
			Type ObjectType
			Ref  string
		}
		switch fields[1] {
		case "blob":
			result = append(result, Item{Type: BLOB, Ref: fields[2]})
		case "tree":
			result = append(result, Item{Type: TREE, Ref: fields[2]})
		default:
			log.Fatalln("ReadTree: unexpected object type:", fields[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading commit object: %v", err)
	}
	return result, nil
}
