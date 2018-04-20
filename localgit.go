package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path"
	"strings"
)

// localGit is a Manager implementation for a local git repo.
// We can't reuse the storeManager with a trivial FS implementation because
// local git repositories are more complicated and some objects could be stored
// in packs and we don't want to have to bother implementing the decompression
// if we can use git-core tools
type localGit struct {
	gitDir string
}

func (lg localGit) ListRefs() ([]Ref, error) {
	// git show-ref
	out, err := exec.Command("git", "show-ref").Output()
	if err != nil {
		return nil, fmt.Errorf("ListRefs: %v", err)
	}

	results := []Ref{}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		var value, name string
		if _, err := fmt.Sscanf(line, "%s %s", &value, &name); err != nil {
			log.Fatalf(
				"git show-ref output not in expected format: \"%s\": %v",
				line, err,
			)
		}
		results = append(results, Ref{Name: name, Value: value})
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("ListRefs: scanning results of git show-ref: %v", err)
	}
	return results, nil
}

func (lg localGit) ReadRef(name string) (string, error) {
	// read file
	bs, err := ioutil.ReadFile(path.Join(lg.gitDir, name))
	if err != nil {
		//var op errors.Op = "localGit.ReadRef"
		//var param errors.Param = name
		//return "", errors.E(op,err)
		return "", fmt.Errorf("ReadRef: reading ref %s: %v", name, err)
	}
	return strings.TrimRight(string(bs), "\n"), nil
}

func (lg localGit) WriteRef(ref Ref) error {
	fullPath := path.Join(lg.gitDir, ref.Value)
	data := []byte(ref.Value + "\n")
	// 0666 is before umask
	if err := ioutil.WriteFile(fullPath, data, 0666); err != nil {
		return fmt.Errorf("WriteRef: writing %s: %v", ref.Name, err)
	}
	return nil
}

func (lg localGit) ReadObject(sha string, contents io.Writer) error {
	return errors.New("not implemented")
}

func (lg localGit) ReadRaw(sha string, contents io.Writer) error {
	return errors.New("not implemented")
}

func (lg localGit) WriteRaw(sha string, contents io.Reader) error {
	return errors.New("not implemented")
}
