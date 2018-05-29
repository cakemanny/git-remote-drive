package store

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	query "github.com/cakemanny/git-remote-drive/store/query"
	drive "google.golang.org/api/drive/v3"
	googleapi "google.golang.org/api/googleapi"
)

// Some fake
type FakeFile struct {
	Name      string
	Parents   []string
	ID        string
	IsFolder  bool
	IsTrashed bool
}

var fsRoot string = "appDataFolder"

// implement ID as unix path for simplicity
var fakeFiles = []FakeFile{
	{"bin", []string{fsRoot}, "/bin", true, false},
	{"bash", []string{"/bin"}, "/bin/bash", false, false},
	{"etc", []string{fsRoot}, "/etc", true, false},
	{"hosts", []string{"/etc"}, "/etc/hosts", false, false},
	{"home", []string{fsRoot}, "/home", true, false},
	{"deleteduser", []string{"/home"}, "/home/deleteduser", true, true},
}

func evalQ(q string, file FakeFile) bool {
	r := strings.NewReader(q)
	expr, err := query.NewParser(r).Parse()
	if err != nil {
		panic(err)
	}

	for _, and := range expr.Ands {
		if evalAnd(and, file) {
			return true
		}
	}
	return false
}
func evalAnd(and query.And, file FakeFile) bool {
	for _, test := range and.Tests {
		if !evalTest(test, file) {
			return false
		}
	}
	return true
}
func evalTest(test query.Test, file FakeFile) bool {
	switch test.Op {
	case query.EQUALS:
		d1 := evalDatum(test.Lhs, file)
		d2 := evalDatum(test.Rhs, file)
		return d1 == d2
	case query.IN:
		d1 := evalDatum(test.Lhs, file)
		if d1.Type != query.STRING {
			return false
		}
		d2 := evalCollection(test.Rhs, file)
		for _, value := range d2 {
			if d1.Lit == value {
				return true
			}
		}
		return false
	case query.CONTAINS:
		// TODO
	}
	return false
}
func evalDatum(d query.Datum, f FakeFile) query.Datum {
	switch d.Type {
	case query.IDENT:
		if d.Lit == "name" {
			return query.Datum{query.STRING, f.Name}
		}
		if d.Lit == "trashed" {
			if f.IsTrashed {
				return query.Datum{query.TRUE, "true"}
			} else {
				return query.Datum{query.FALSE, "false"}
			}
		}
		return d
	case query.STRING:
		return d
	case query.TRUE:
		return d
	case query.FALSE:
		return d
	}
	panic(fmt.Sprintf("unexpected datum type: %v", d.Type))
}
func evalCollection(d query.Datum, f FakeFile) []string {
	if d.Type == query.IDENT && d.Lit == "parents" {
		return f.Parents
	}
	return nil
}

// want to mock out the drive.FileService in our driveAPIClient.srv

type fakeFilesService struct{}

func (fakeFilesService) List() FilesListCall {
	return fakeFilesListCall{}
}
func (fakeFilesService) Get(fileId string) FilesGetCall {
	return fakeFilesGetCall{}
}

type fakeFilesListCall struct {
	spaces   string
	pageSize int64
	q        string
	fields   []string
	opts     []googleapi.CallOption
}

func (lc fakeFilesListCall) Spaces(spaces string) FilesListCall {
	lc.spaces = spaces
	return &lc
}
func (lc fakeFilesListCall) PageSize(pageSize int64) FilesListCall {
	lc.pageSize = pageSize
	return &lc
}
func (lc fakeFilesListCall) Q(q string) FilesListCall {
	lc.q = q
	return &lc
}
func (lc fakeFilesListCall) Fields(s ...googleapi.Field) FilesListCall {
	lc.fields = nil
	for _, x := range s {
		lc.fields = append(lc.fields, string(x))
	}
	return &lc
}
func (lc fakeFilesListCall) Do(opts ...googleapi.CallOption) (*drive.FileList, error) {
	lc.opts = opts

	var results []*drive.File
	for _, fakeFile := range fakeFiles {
		if evalQ(lc.q, fakeFile) {
			var mimeType string
			if fakeFile.IsFolder {
				mimeType = "application/vnd.google-apps.folder"
			} else {
				mimeType = "application/octet-stream"
			}

			results = append(results, &drive.File{
				Name:     fakeFile.Name,
				Trashed:  fakeFile.IsTrashed,
				Parents:  fakeFile.Parents,
				Id:       fakeFile.ID,
				MimeType: mimeType,
			})
		}
	}

	return &drive.FileList{
		Files:          results,
		Kind:           "drive#fileList",
		ServerResponse: googleapi.ServerResponse{HTTPStatusCode: 200},
	}, nil
}

func (fakeFilesService) Create(*drive.File) FilesCreateCall {
	return fakeFilesCreateCall{}
}

type fakeFilesCreateCall struct {
	reader       io.Reader
	mediaOptions []googleapi.MediaOption
	fields       []string
	callOptions  []googleapi.CallOption
	doCalled     int
}

func (cc fakeFilesCreateCall) Fields(s ...googleapi.Field) FilesCreateCall {
	for _, field := range s {
		cc.fields = append(cc.fields, string(field))
	}
	return cc
}
func (cc fakeFilesCreateCall) Media(r io.Reader, options ...googleapi.MediaOption) FilesCreateCall {
	for _, option := range options {
		cc.mediaOptions = append(cc.mediaOptions, option)
	}
	cc.reader = r
	return cc
}
func (cc fakeFilesCreateCall) Do(opts ...googleapi.CallOption) (*drive.File, error) {
	for _, opt := range opts {
		cc.callOptions = append(cc.callOptions, opt)
	}
	cc.doCalled++
	return nil, nil
}

type fakeFilesGetCall struct {
	callOptions []googleapi.CallOption
}

func (gc fakeFilesGetCall) Download(opts ...googleapi.CallOption) (*http.Response, error) {
	for _, opt := range opts {
		gc.callOptions = append(gc.callOptions, opt)
	}
	return nil, nil
}

func TestGetID(t *testing.T) {
	srv := &Service{fakeFilesService{}}
	client := driveAPIClient{srv, map[[2]string]string{}}

	id, err := client.GetID("bash", "/bin")

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if id != "/bin/bash" {
		t.Error("expected:", "/bin/bash", "actual:", id)
	}
}
