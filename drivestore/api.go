package drivestore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	paths "path"
	"strings"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
)

var (
	tokenPath  = os.ExpandEnv("$HOME/.git-remote-drive.json")
	secretPath = os.ExpandEnv("$HOME/.git-remote-drive.secret")

	// So we get compile-time errors when we fat-finger this
	appDataFolder = "appDataFolder"

	// ErrNotFound is returned when a file cannot be found
	ErrNotFound = errors.New("not found")
)

type File struct {
    IsFolder bool
    Name string
}

type ReadOnlyStore interface {
	Read(path string, contents io.Writer) error
    List(path string) ([]File, error)
}

// SimpleFileStore is a simplied interface over a file store
type SimpleFileStore interface {
	Create(path string, contents io.Reader) error
	Read(path string, contents io.Writer) error
	Update(path string, contents io.Reader) error
	Delete(path string) error
    List(path string) ([]File, error)
}

// DriveAPIClient is an implementation of a file store using Google Drive.
type DriveAPIClient struct {
	srv *drive.Service
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tok, err := tokenFromFile(tokenPath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenPath, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieve a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

// NewClient runs the outh flow if necessary and builds an authenticated
// Google Drive client
func NewClient() DriveAPIClient {
	b, err := ioutil.ReadFile(secretPath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.
	config, err := google.ConfigFromJSON(b, drive.DriveAppdataScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	srv, err := drive.New(getClient(config))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	return DriveAPIClient{srv}
}

// MkDir creates a folder recursively (think mkdir -p) and returns the
// file id of the created directory.
func (client DriveAPIClient) MkDir(path string) (string, error) {
	parentID, err := GetIDRecursive(client, path)
	if err == ErrNotFound {
		parentID, err = client.MkDir(paths.Dir(path))
	}
	if err != nil {
		return "", err
	}
	info := drive.File{
		Name:     paths.Base(path),
		Parents:  []string{parentID},
		MimeType: "application/vnd.google-apps.folder",
	}
	result, err := client.srv.Files.Create(&info).Fields("id").Do()
	if err != nil {
		return "", err
	}
	return result.Id, nil
}

// GetID returns the file ID of a file with the given name in the
// folder with ID parentID
func (client DriveAPIClient) GetID(name string, parentID string) (string, error) {
	if parentID == "" {
		parentID = appDataFolder
	}
	// prove we won't break the query string
	replacer := strings.NewReplacer("'", "\\'", "\\", "\\\\")
	if strings.ContainsAny(name, "'\\") {
		name = replacer.Replace(name)
	}
	if strings.ContainsAny(parentID, "'\\") {
		parentID = replacer.Replace(parentID)
	}
	r, err := client.srv.Files.List().Spaces(appDataFolder).
		PageSize(2).
		Q(fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false", name, parentID)).
		Fields("files(id)").
		Do()
	if err != nil {
		return "", err
	}
	if len(r.Files) == 0 {
		return "", ErrNotFound
	}
	// Should we error if there are more than one result?
	if len(r.Files) > 1 {
		log.Printf("Warning: \n")
	}
	return r.Files[0].Id, nil
}

// define this to allow us to unit test the recursive ID getter
type idGetter interface {
	GetID(name string, parentID string) (string, error)
}

// GetIDRecursive navigates up the directory tree
func GetIDRecursive(client idGetter, path string) (string, error) {
	parentPath, name := paths.Dir(path), paths.Base(path)
	if parentPath == "." {
		// No more parent IDs to get
		return client.GetID(name, "")
	}
	// Not root level. Find in parent
	parentID, err := GetIDRecursive(client, parentPath)
	if err != nil {
		return "", err
	}
	return client.GetID(name, parentID)
}

// TODO: implement caching around ID getter

// Create creates a file in the user's Google Drive
func (client DriveAPIClient) Create(path string, contents io.Reader) error {
	// 1. Resolve the parent path - create if not exists
	parentPath := paths.Dir(path)
	parentID, err := func() (string, error) {
		if parentPath == "." || parentPath == "/" {
			return appDataFolder, nil
		}
		return GetIDRecursive(client, parentPath)
	}()
	if err == ErrNotFound {
		parentID, err = client.MkDir(parentPath)
		if err != nil {
			err = fmt.Errorf("error creating directory %s: %s", parentPath, err)
		}
	}
	if err != nil {
		return err
	}

	filename := paths.Base(path)
	file := drive.File{Name: filename, Parents: []string{parentID}}

	result, err := client.srv.Files.Create(&file).Fields("id").
		Media(contents).Do()
	if err != nil {
		return err
	}
	fmt.Printf("Created file, ID: %s\n", result.Id)
	return nil
}

func (client DriveAPIClient) Read(path string, contents io.Writer) error {

	return errors.New("not implemented")
}

func (client DriveAPIClient) List(path string) ([]File, error) {

    return nil, errors.New("not implemented")
}

func (client DriveAPIClient) Delete(path string) error {
    return errors.New("not implemented")
}

func (client DriveAPIClient) Update(path string, contents io.Reader) error {
    return errors.New("not implemented")
}
