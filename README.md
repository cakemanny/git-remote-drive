Git Remote Drive
================
We are not at a working version yet

Progress to initial working first draft:
- [x] `capabilities`
- [x] `list` and `list for-push`
- [x] `option`
- [ ] `fetch`
- [ ] `push`

Things to go in a v1:
- [ ] `--init` or `--auth` option to run outh2 flow
- [ ] loopback interface redirect for oauth2 browser, instead of copy/paste
- [ ] caching layer to reduce naive API calls
- [ ] use go's channels to push or fetch multiple objects concurrently
- [ ] continuous integration

What is it?
-----------
A git remote helper for storing git repositories in Google Drive using the
Google Drive API.

```shell
$ git remote add drive drive://git/my-project.git
$ git push drive master
...
$ git pull drive/master
```

Requirements
------------
* [go](https://golang.org/) 1.10.1 or later

Getting Started
---------------
### Setting up
* Create a project in the
  [Google API Console](https://console.developers.google.com/)
* Enable the Google Drive API in your project:
  - Go to the [API Library page](https://console.developers.google.com/apis/library)
  - Find **Google Drive API** and enable it for your project.
* Create a client secret key:
  - Go to the [Credentials page](https://console.developers.google.com/apis/credentials)
  - From the **Create credentials** dropdown select **OAuth client ID**
  - Select **Other** as the application type
  - Name the client something like **git-remote-drive-client**
  - Click **Create**
  - Dismiss the popup with the client ID and secret
* Download the credentials file and store it as ~/.git-remote-drive.secret
* Change the permissions of the secret file to make them private

On a unix or unix-like system:

```shell
$ chmod 0600 ~/.git-remote-drive.secret
```

<!-- TODO: work out instructions for Windows -->
<!-- TODO: double check this works on Linux -->


We assume you have a `GOPATH` set up and the `go` tool on your `PATH`.
Instructions for this can be found at <https://golang.org/doc/code.html>

Download and install `git-remote-drive`:

```shell
$ go get github.com/cakemanny/git-remote-drive
$ go install github.com/cakemanny/git-remote-drive
```
<!-- TODO: double check that this is how you install go stuff.. -->

* Make sure `$GOPATH/bin` is on your path
* Or link the binary into a folder that is, e.g:

```shell
$ ln -s $GOPATH/bin/git-remote-drive ~/bin/.
```

Run the outh2 flow to give the app access to your Google Drive account.

```shell
$ git-remote-drive --init
```

You will be prompted to a follow go to a URL in your browser. Go to the URL
authorise the app and then paste the token back into the prompt.
<!-- TODO: add some of the output -->

A token file will be saved in your home directory: `~/.git-remote-drive.json`

### Create a test repository
Create a git repository with at least one commit

```shell
mkdir test-project && cd test-project
echo "This is a test project" > README.txt
git add README.txt
git commit -m "initial commit"
```

Then add the remote:

```shell
$ git remote add origin drive://path/to/repos/test-project.git
```

You should now be able to push and fetch from the remote:

```shell
$ git push origin master
$ git fetch origin
```
<!-- TODO: add output here! -->

Workflow
--------
TODO: write about cloning

Storage Details
---------------
TODO: write about which parts of existing project layout we use

Potential Problems
------------------
Google Drive allows multiple files or directories with the same path. Git
objects are content addressable, so if duplicates are created this should not
cause any problems - they will be identical and are never updated.

On the other hand, refs are updated and created. Concurrent pushes may cause
duplicate refs to be created.

How does Git Remote Drive solve this problem?
<!-- TODO: write about our solution -->

Inspiration and Reason
----------------------
This was inspired by having used
<https://github.com/anishathalye/git-remote-dropbox>
for a few years, and thinking it would be interesting to see if other cloud
user file storage providers could be used. I also needed to learn go, so
what better a marrying?

Since this is my first project in go, please feel free to comment on the
implementation, and suggest improvements, perhaps by pull request.

<!-- Do we want this? -->
<footer><small>Copyright &copy; 2018 Daniel Golding - MIT License</small></footer>

