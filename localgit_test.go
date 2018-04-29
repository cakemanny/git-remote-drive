package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestLocalGit(t *testing.T) {
	testLocalGit(t, true)
	testLocalGit(t, false)
}

func testLocalGit(t *testing.T, gc bool) {
	startDir := os.Getenv("PWD")

	tmpDir, err := ioutil.TempDir("", "tmprepo")
	defer os.RemoveAll(tmpDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal("cd", tmpDir, err)
	}
	defer os.Chdir(startDir)

	script := "git init && " +
		"echo hi > test.txt && " +
		"git add test.txt && " +
		"git commit -m 'initial commit' " +
		" --author='A U Thor <author@example.com>' " +
		" --date='1970-01-01 00:00:00'"
	if gc {
		// gc so we test with packs instead of loose objects
		script += " && git gc"
	}
	if err := exec.Command("/bin/sh", "-c", script).Run(); err != nil {
		t.Fatal("git init add and commit:", err)
	}

	var lg = localGit{gitDir: ".git"}

	t.Run("TestReadRef", func(t *testing.T) {
		refs, err := lg.ListRefs()
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		expectedName := "refs/heads/master"

		if len(refs) != 1 || refs[0].Name != expectedName || len(refs[0].Value) != 40 {
			t.Error(
				"expected:", expectedName,
				"actual:", refs,
			)
		}
	})

	const hisha string = "45b983be36b73c0788dc9cbcb76cbb80fc7bb057"
	t.Run("TestReadObject", func(t *testing.T) {
		var sb strings.Builder
		err = lg.ReadObject(hisha, &sb)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		expected := "hi\n"
		if sb.String() != expected {
			t.Errorf(`expected: "%s", actual: "%s"`, expected, sb.String())
		}
	})
	t.Run("TestReadRaw", func(t *testing.T) {
		var buf bytes.Buffer
		err := lg.ReadRaw(hisha, &buf)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		actualSha, err := sha1Bytes(buf.Bytes())
		if err != nil {
			t.Error("error hashing contents: ", err)
		}
		if actualSha != hisha {
			t.Error("expected-sha1:", hisha, "actual-sha1:", actualSha)
		}
	})
	t.Run("TestReadPacked", func(t *testing.T) {
		var buf bytes.Buffer
		err := lg.readPacked(hisha, &buf)
		if err != nil {
			t.Fatal("unexpected error", err)
		}
		actualSha, err := sha1Bytes(buf.Bytes())
		if err != nil {
			t.Error("error hashing contents: ", err)
		}
		if actualSha != hisha {
			t.Error("expected-sha1:", hisha, "actual-sha1:", actualSha)
		}
	})
	t.Run("TestGetType", func(t *testing.T) {
		refType, err := lg.GetType(hisha)
		if err != nil {
			t.Error("unexpected error", err)
		}
		if refType != "blob" {
			t.Errorf(`expected: "blob" actual: "%s"`, refType)
		}
	})
}
