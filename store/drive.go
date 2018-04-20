package store

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
)

// driveAPIClient is an implementation of a SimpleFileStore using Google Drive.
type driveAPIClient struct {
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
func NewClient() SimpleFileStore {
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
	return driveAPIClient{srv}
}

// MkDir creates a folder recursively (think mkdir -p) and returns the
// file id of the created directory.
func (client driveAPIClient) MkDir(path string) (string, error) {
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
func (client driveAPIClient) GetID(name string, parentID string) (string, error) {
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
		Q(fmt.Sprintf("name = '%s' and '%s' in parents and trashed = false",
			name, parentID)).
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
		log.Printf("warning: more than one \"%s\"\n", name)
	}
	return r.Files[0].Id, nil
}

// Create creates a file in the user's Google Drive
func (client driveAPIClient) Create(path string, contents io.Reader) error {
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

func (client driveAPIClient) Read(path string, contents io.Writer) error {
	fileID, err := func() (string, error) {
		if path == "/" || path == "" {
			return appDataFolder, nil
		}
		return GetIDRecursive(client, path)
	}()
	if err != nil {
		return err
	}
	r, err := client.srv.Files.Get(fileID).Download()
	if err != nil {
		return fmt.Errorf("error requesting \"%s\": %v", path, err)
	}
	_, err = io.Copy(contents, r.Body)
	if err != nil {
		return fmt.Errorf("while reading \"%s\": %v", path, err)
	}
	// Not sure we care about the close error is we managed to read all
	// successfully
	r.Body.Close()
	return nil
}

func (client driveAPIClient) List(path string) ([]File, error) {
	folderID, err := func() (string, error) {
		if path == "/" || path == "" {
			return appDataFolder, nil
		}
		return GetIDRecursive(client, path)
	}()

	// Would we need to escape folderID ever?
	r, err := client.srv.Files.List().Spaces(appDataFolder).
		Q(fmt.Sprintf("'%s' in parents and trashed = false", folderID)).
		Fields("files(id,name,mimeType)").
		Do()
	if err != nil {
		return nil, err
	}

	results := make([]File, len(r.Files))
	for i, f := range r.Files {
		results[i] = File{
			IsFolder: (f.MimeType == "application/vnd.google-apps.folder"),
			Name:     f.Name,
		}
	}
	return results, nil
}

func (client driveAPIClient) Delete(path string) error {
	// not sure this is needed
	return errors.New("not implemented")
}

func (client driveAPIClient) Update(path string, contents io.Reader) error {
	return errors.New("not implemented")
}