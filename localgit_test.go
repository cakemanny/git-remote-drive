package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestLocalListRefs(t *testing.T) {
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
	if err := exec.Command("/bin/sh", "-c", script).Run(); err != nil {
		t.Fatal("git init add and commit:", err)
	}

	var lg Manager = localGit{gitDir: ".git"}
	refs, err := lg.ListRefs()
	if err != nil {
		t.Error("unexpected error", err)
	}

	expectedName := "refs/heads/master"

	if len(refs) != 1 || refs[0].Name != expectedName || len(refs[0].Value) != 40 {
		t.Error(
			"expected:", expectedName,
			"actual:", refs,
		)
	}
}
