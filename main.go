package main

import (
    "bufio"
    "fmt"
    "io"
    "log"
    "os"
    "strings"

    "github.com/cakemanny/git-remote-drive/drivestore"
)

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

    var fileStore drivestore.ReadOnlyStore = drivestore.NewClient()
    var lister RefLister = storeManager{
        strings.TrimPrefix(driveUrl, "drive://"),
        fileStore,
    }

    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        line := scanner.Text()
        log.Printf(line)
        dispatch(line, os.Stdout, lister)
    }
    if err := scanner.Err(); err != nil {
        log.Printf("reading standard input: %s", err)
    }
}

func dispatch(line string, out io.Writer, lister RefLister) {
    switch line {
    case "capabilities":
        fmt.Fprintln(out, "push") // FIXME: shouldn't have this for RO store
        fmt.Fprintln(out, "fetch")
        fmt.Fprintln(out, "option")
        fmt.Fprintln(out)
    case "list":
        listRefs(out, lister)
    case "list for-push":
        listRefs(out, lister)
    default:
        if strings.HasPrefix(line, "option ") {
            fields := strings.Fields(line)
            if len(fields) > 1 && fields[1] == "verbosity" {
                if len(fields) < 3 {
                    fmt.Fprintln(out, "err invalid verbosity")
                }
                var verbosity int
                _, err := fmt.Sscanf(fields[2], "%d", &verbosity)
                if err != nil {
                    log.Printf("error reading verbosity value \"%s\", %v",fields[2], err)
                    fmt.Fprintln(out, "err invalid verbosity")
                    return
                }
                fmt.Fprintln(out, "ok")
                return
            }
            fmt.Fprintln(out, "unsupported")
            return
        }
        if strings.HasPrefix(line, "fetch ") {

        }
        if strings.HasPrefix(line, "push ") {
        }
    }
}

func listRefs(out io.Writer, lister RefLister) {
    refs, err := lister.ListRefs()
    if err != nil {
        log.Fatalf("%v", err)
    }
    for _, ref := range refs {
        fmt.Fprintf(out, "%s %s\n", ref.Value, ref.Name)
    }
    // End with blank line
    fmt.Fprintln(out)
}



