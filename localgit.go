package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
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
				`git show-ref output not in expected format: "%s": %v`,
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
	// Maybe we should be using git-rev-parse?
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
	// Maybe we should be using git-update-ref?
	fullPath := path.Join(lg.gitDir, ref.Value)
	data := []byte(ref.Value + "\n")
	// 0666 is before umask
	if err := ioutil.WriteFile(fullPath, data, 0666); err != nil {
		return fmt.Errorf("WriteRef: writing %s: %v", ref.Name, err)
	}
	return nil
}

func (lg localGit) GetType(sha string) (string, error) {
	if len(sha) != 40 {
		return "", fmt.Errorf(`invalid sha: "%s"`, sha)
	}
	res, err := exec.Command("git", "cat-file", "-t", sha).Output()
	if err != nil {
		return "", fmt.Errorf("getting type of %s", sha)
	}
	return strings.TrimRight(string(res), "\n"), nil
}

func (lg localGit) ReadObject(sha string, contents io.Writer) error {
	cmd := exec.Command("git", "cat-file", "-p", sha)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("redirecting stdout: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting git cat-file: %v", err)
	}
	io.Copy(contents, stdout)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git cat-file: %v", err)
	}
	return nil
}

func (lg localGit) ReadRaw(sha string, contents io.Writer) error {
	if len(sha) != 40 {
		return fmt.Errorf(`invalid sha: "%s"`, sha)
	}
	fullPath := path.Join(lg.gitDir, "objects", sha[:2], sha[2:])
	file, err := os.Open(fullPath)
	if err != nil {
		// If the file is not loose it may well be packed
		pathErr, ok := err.(*os.PathError)
		if ok && pathErr.Err.Error() == "no such file or directory" {
			return lg.readPacked(sha, contents)
		}
		// Want to check if is filenotfound
		return fmt.Errorf(`opening "%s": %v`, fullPath, err)
	}
	defer file.Close()
	_, err = io.Copy(contents, file)
	if err != nil {
		return fmt.Errorf(`reading "%s": %v`, fullPath, err)
	}
	return nil
}

func (lg localGit) readPacked(sha string, contents io.Writer) error {
	// Idea (explained in shell):
	//   type=$(git cat-file -t $sha)
	//   size=$(git cat-file -s $sha)
	//   content=$(git cat-file $type $sha)
	//   echo -nE "$type $size"$'\0'"$content" | zlib-flate -compress

	objectType, err := lg.GetType(sha)
	if err != nil {
		return err
	}
	sizeRes, err := exec.Command("git", "cat-file", "-s", sha).Output()
	if err != nil {
		return fmt.Errorf("getting size of object %s: %v", sha, err)
	}
	size := strings.TrimRight(string(sizeRes), "\n")

	content, err := exec.Command("git", "cat-file", objectType, sha).Output()
	if err != nil {
		return fmt.Errorf("getting content of object %s: %v", sha, err)
	}

	zlibWrtr := zlib.NewWriter(contents)
	defer zlibWrtr.Close()
	if _, err = zlibWrtr.Write([]byte(objectType + " " + size)); err != nil {
		return err
	}
	if _, err = zlibWrtr.Write([]byte{0}); err != nil {
		return err
	}
	if _, err = zlibWrtr.Write(content); err != nil {
		return err
	}
	return nil
}

func (lg localGit) WriteRaw(sha string, contents io.Reader) error {
	return errors.New("not implemented")
}
