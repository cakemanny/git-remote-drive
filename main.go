package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cakemanny/git-remote-drive/store"
)

var options struct {
	verbosity  int
	followtags bool
}

/*
main gets called by git using one of the following command lines
if the url is of the form drive://<rest-of-url>

    $ git-remote-drive <remote> drive://<path>
    $ git-remote-drive drive://<path> drive://<path>

If the URL is of the form drive::<path>

    $ git-remote-drive <remote> <path>
    $ git-remote-drive <path> <path>

*/
func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Not enough command line arguments. Was: %v", os.Args)
	}
	remoteName := os.Args[1]
	_ = remoteName
	driveUrl := os.Args[2]
	_ = driveUrl

	var fileStore store.SimpleFileStore = store.NewClient()
	var manager Manager = storeManager{
		strings.TrimPrefix(driveUrl, "drive://"),
		fileStore,
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf(line)
		dispatch(line, os.Stdout, manager)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("reading standard input: %s", err)
	}
	log.Printf("exiting gracefully")
}

func dispatch(line string, out io.Writer, manager Manager) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		log.Println("warning: command was only whitespace")
		return
	}
	command := fields[0]
	switch command {
	case "capabilities":
		fmt.Fprintln(out, "push")
		fmt.Fprintln(out, "fetch")
		fmt.Fprintln(out, "option")
		fmt.Fprintln(out)
	case "list":
		// Could be "list" or "list for-push" - same result either way
		listRefs(out, manager)
	case "option":
		// all options are of the format
		// option <setting> <value>...
		if len(fields) < 3 {
			fmt.Fprintln(out, "err invalid option command")
			return
		}
		command, value1 := fields[1], fields[2]
		switch command {
		case "verbosity":
			var v int
			if _, err := fmt.Sscanf(value1, "%d", &v); err != nil {
				log.Printf("error reading verbosity value \"%s\", %v",
					value1, err)
				fmt.Fprintln(out, "err invalid verbosity")
				return
			}
			options.verbosity = v
			fmt.Fprintln(out, "ok")
		case "followtags":
			if value1 != "true" && value1 != "false" {
				fmt.Fprintln(out, "err invalid followtags")
				return
			}
			options.followtags = (value1 == "true")
			fmt.Fprintln(out, "ok")
		default:
			fmt.Fprintln(out, "unsupported")
		}
	case "fetch":
	case "push": // push refs/heads/master:refs/heads/master
		// ok looks like we need to do all the hard work

		// Idea: find out local and remote commits.
		//       work backwards from local adding all reachable objects to a
		//       set of objects to send.
		//		 ?- if we hit the remote commit, stop following that branch.
		//		 ?- if we hit a parent of remote, stop following that branch.
		//		 work from remote commit backwards removing all reachable
		//		 objects from the set.
		//		 raw copy all the objects.
		//		 update the remote ref.
		//

		// Space and colons are invalid branch names - so we are all good with
		// this logic
		if strings.Count(fields[1], ":") != 1 {
			fmt.Fprintln(out, "error")
			fmt.Fprintln(out)
			return
		}
		localRefName, remoteRefName := func() (string, string) {
			x := strings.SplitN(fields[1], ":", 2)
			return x[0], x[1]
		}()

		// TODO: init me
		var localStore store.SimpleFileStore

		var localManager Manager = storeManager{
			basePath: os.ExpandEnv("$GIT_DIR"),
			store:    localStore,
		}

		localRef, err := localManager.ReadRef(localRefName)
		if err != nil {
			log.Fatalln(err)
		}
		remoteRef, err := manager.ReadRef(remoteRefName)
		if err != nil {
			log.Fatalln(err)
		}

		var _ = localRef
		var _ = remoteRef

	default:
	}
}

func listRefs(out io.Writer, lister RefLister) {
	refs, err := lister.ListRefs()
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(refs) == 0 {
		log.Printf("warning: no remote refs found")
	}
	for _, ref := range refs {
		fmt.Fprintf(out, "%s %s\n", ref.Value, ref.Name)
	}
	// End with blank line
	fmt.Fprintln(out)
}
