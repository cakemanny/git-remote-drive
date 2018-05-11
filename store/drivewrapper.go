package store

import (
	"io"
	"net/http"

	drive "google.golang.org/api/drive/v3"
	googleapi "google.golang.org/api/googleapi"
)

type Service struct {
	Files FilesService
}

type FilesService interface {
	//Copy(string, *drive.File) *drive.FilesCopyCall
	Create(*drive.File) FilesCreateCall
	//Delete(string) *drive.FilesDeleteCall
	//EmptyTrash() *drive.FilesEmptyTrashCall
	//Export(string, string) *drive.FilesExportCall
	//GenerateIds() *drive.FilesGenerateIdsCall
	Get(string) FilesGetCall
	List() FilesListCall
	//Update(string, *drive.File) *drive.FilesUpdateCall
	//Watch(string, *drive.Channel) *drive.FilesWatchCall
}

type FilesCreateCall interface {
	Do(opts ...googleapi.CallOption) (*drive.File, error)
	Fields(s ...googleapi.Field) FilesCreateCall
	Media(r io.Reader, options ...googleapi.MediaOption) FilesCreateCall
	//IgnoreDefaultVisibility
	//KeepRevisionForever
	//OcrLanguage
	//SupportsTeamDrives
	//UseContentAsIndexableText
	//ResumableMedia
	//ProgressUpdater
	//Context
	//Header
}

type FilesGetCall interface {
	Download(opts ...googleapi.CallOption) (*http.Response, error)
}

type FilesListCall interface {
	Spaces(spaces string) FilesListCall
	PageSize(pageSize int64) FilesListCall
	Q(q string) FilesListCall
	Fields(s ...googleapi.Field) FilesListCall
	Do(opts ...googleapi.CallOption) (*drive.FileList, error)
}

// A wrapper around the google drive api drive.FilesService that implements
// our interfaces
type filesServiceWrapper struct {
	filesServices *drive.FilesService
}
type filesCreateWrapper struct {
	filesCreate *drive.FilesCreateCall
}
type filesGetWrapper struct {
	filesGet *drive.FilesGetCall
}
type filesListWrapper struct {
	filesList *drive.FilesListCall
}

func (wrapper filesServiceWrapper) Create(file *drive.File) FilesCreateCall {
	return filesCreateWrapper{wrapper.filesServices.Create(file)}
}
func (wrapper filesServiceWrapper) Get(fileId string) FilesGetCall {
	return filesGetWrapper{wrapper.filesServices.Get(fileId)}
}
func (wrapper filesServiceWrapper) List() FilesListCall {
	return filesListWrapper{wrapper.filesServices.List()}
}

func (wrapper filesCreateWrapper) Do(opts ...googleapi.CallOption) (*drive.File, error) {
	return wrapper.filesCreate.Do(opts...)
}
func (wrapper filesCreateWrapper) Fields(s ...googleapi.Field) FilesCreateCall {
	return filesCreateWrapper{wrapper.filesCreate.Fields(s...)}
}
func (wrapper filesCreateWrapper) Media(r io.Reader, options ...googleapi.MediaOption) FilesCreateCall {
	return filesCreateWrapper{wrapper.filesCreate.Media(r, options...)}
}

func (wrapper filesGetWrapper) Download(opts ...googleapi.CallOption) (*http.Response, error) {
	return wrapper.filesGet.Download(opts...)
}
func (wrapper filesListWrapper) Spaces(spaces string) FilesListCall {
	return filesListWrapper{wrapper.filesList.Spaces(spaces)}
}
func (wrapper filesListWrapper) PageSize(pageSize int64) FilesListCall {
	return filesListWrapper{wrapper.filesList.PageSize(pageSize)}
}
func (wrapper filesListWrapper) Q(q string) FilesListCall {
	return filesListWrapper{wrapper.filesList.Q(q)}
}
func (wrapper filesListWrapper) Fields(s ...googleapi.Field) FilesListCall {
	return filesListWrapper{wrapper.filesList.Fields(s...)}
}
func (wrapper filesListWrapper) Do(opts ...googleapi.CallOption) (*drive.FileList, error) {
	return wrapper.filesList.Do(opts...)
}
