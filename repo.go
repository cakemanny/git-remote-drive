package main

import (
    "fmt"
    "io"
    "path"
    "strings"

    ds "github.com/cakemanny/git-remote-drive/drivestore"
)

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
    store ds.ReadOnlyStore
}

func (m storeManager) ListRefs() ([]Ref, error) {
    var results []Ref
    pathToRefs := path.Join(m.basePath, "refs")

    onFile := func (p string) error {
        var sha1val strings.Builder
        if err := m.store.Read(p, &sha1val); err != nil {
            return fmt.Errorf("error reading \"%s\" in remote store: %v", p, err)
        }
        name := strings.TrimPrefix(strings.TrimPrefix(p, m.basePath), "/")
        results = append(results, Ref{ sha1val.String(), name })
        return nil
    }

    var walk func(string) error
    walk = func(baseDir string) error {
        list, err := m.store.List(baseDir)
        if err == ds.ErrNotFound {
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

