package main

import (
    "io"
	"testing"
    "strings"

    "github.com/cakemanny/git-remote-drive/drivestore"
)

type fakeReadStore struct {}
var fakeStore fakeReadStore

func (fakeReadStore) Read(path string, contents io.Writer) error {
    if path == "refs/heads/master" {
        _, err :=io.WriteString(
            contents, "c5d2d737af4b6203aa37ca2ca13476624d11f4ee",
        )
        return err
    }
    if path == "objects/c5/d2d737af4b6203aa37ca2ca13476624d11f4ee" {
        _, err := io.WriteString(contents, "do I have to put real data here?")
        return err
    }
    return drivestore.ErrNotFound
}

func (fakeReadStore) List(path string) ([]drivestore.File, error) {
    if path == "" {
        return []drivestore.File{ { true, "objects"}, { true, "refs" } }, nil
    }
    if path == "refs" {
        return []drivestore.File{ { true, "heads" } }, nil
    }
    if path == "refs/heads" {
        return []drivestore.File{ { false, "master" } }, nil
    }
    if path == "objects" {
        return []drivestore.File{ { true, "c5" } }, nil
    }
    if path == "objects/c5" {
        return []drivestore.File{ { false, "d2d737af4b6203aa37ca2ca13476624d11f4ee" } }, nil
    }
    return []drivestore.File{}, nil
}


func TestDispatch(t *testing.T) {
    matrix := []struct {
        command, expected string
    }{
        { "capabilities", "push\nfetch\noption\n\n" },
        { "option verbosity 0", "ok\n" },
        { "option verbosity not a number", "err invalid verbosity\n" },
        { "option boobity bob", "unsupported\n" },
    }
    for _, v := range matrix {
        var out strings.Builder
        dispatch(v.command, &out, storeManager{"", &fakeStore})
        result := out.String()
        if result != v.expected {
            t.Error("command:", v.command, "expected:", v.expected,
                "actual:", result)
        }
    }
}

func TestListRefs(t *testing.T) {

    expected := "c5d2d737af4b6203aa37ca2ca13476624d11f4ee refs/heads/master\n\n"

    m := storeManager{ "", &fakeStore }

    var sb strings.Builder
    listRefs(&sb, m)
    result := sb.String()
    if result != expected {
        t.Errorf("command:list\n\texpected:\"%s\",\n\tactual: \"%s\"",
                expected, result)
    }
}


