package store

import (
	drive "google.golang.org/api/drive/v3"
	googleapi "google.golang.org/api/googleapi"
	"testing"
)

// want to mock out the drive.FileService in our driveAPIClient.srv

type fakeFilesService struct{}

func (fakeFilesService) List() FilesListCall {
	return fakeFilesListCall{}
}

type fakeFilesListCall struct {
	spaces   string
	pageSize int64
	q        string
	fields   []string
}

func (lc *fakeFilesListCall) Spaces(spaces string) FilesListCall {
	lc.spaces = spaces
	return lc
}
func (lc *fakeFilesListCall) PageSize(pageSize int64) FilesListCall {
	lc.pageSize = pageSize
	return lc
}
func (lc *fakeFilesListCall) Q(q string) FilesListCall {
	lc.q = q
	return lc
}
func (lc *fakeFilesListCall) Fields(s ...googleapi.Field) FilesListCall {
	return lc
}
func (lc fakeFilesListCall) Do(opts ...googleapi.CallOption) (*drive.FileList, error) {
	return nil, nil
}

func TestGetID(t *testing.T) {
	srv := &Service{fakeFilesService{}}
	client := driveAPIClient{srv, map[[2]string]string{}}

}
