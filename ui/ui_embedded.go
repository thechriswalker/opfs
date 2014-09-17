// +build embedui

package ui

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"
)

//embeddedUI is auto-generated @see `build.go`
func Handler(path *string) http.Handler {
	return http.FileServer(&UI{fs: embeddedUI})
}

var efsError404 = fmt.Errorf("File doesn't exist")

//UI's should be built to expect to be served from `/ui/` not `/`
//so we have 2 implementations that implement http.FileSystem
type EmbeddedHttpFileSystem map[string]*embeddedFile

func (efs EmbeddedHttpFileSystem) Open(name string) (http.File, error) {
	file, ok := efs[name]
	if !ok {
		return nil, efsError404
	}
	if file.reader != nil {
		file.reader.Seek(0, 0)
	}
	return file, nil
}

// http.file interface is like:
// type File interface {
//         io.Closer
//         io.Reader
//         Readdir(count int) ([]os.FileInfo, error)
//         Seek(offset int64, whence int) (int64, error)
//         Stat() (os.FileInfo, error)
// }
type embeddedFile struct {
	reader *bytes.Reader //io.Reader, io.Seeker
	stat   os.FileInfo   //for the os.FileInfo interface
}

//close is a no-op
func (ef *embeddedFile) Close() error { return nil }

//pass read to ef.reader
func (ef *embeddedFile) Read(b []byte) (int, error) {
	return ef.reader.Read(b)
}

//pass Seek to ef.reader
func (ef *embeddedFile) Seek(o int64, w int) (int64, error) {
	return ef.reader.Seek(o, w)
}

//stat is static.
func (ef *embeddedFile) Stat() (os.FileInfo, error) {
	return ef.stat, nil
}

func (ef *embeddedFile) Readdir(count int) ([]os.FileInfo, error) {
	//we have no directories! they 404!
	//so this should never happen, but return the error anyway.
	return nil, fmt.Errorf("No Directories in EmbeddedHttpFileSystem")
}

//embeddedFileInfo has the os.FileInfo interface:
// type FileInfo interface {
//         Name() string       // base name of the file
//         Size() int64        // length in bytes for regular files; system-dependent for others
//         Mode() FileMode     // file mode bits
//         ModTime() time.Time // modification time
//         IsDir() bool        // abbreviation for Mode().IsDir()
//         Sys() interface{}   // underlying data source (can return nil)
// }
type embeddedFileInfo struct {
	size int64
	name string
	time time.Time
}

func (e *embeddedFileInfo) Name() string       { return e.name }
func (e *embeddedFileInfo) Size() int64        { return e.size }
func (e *embeddedFileInfo) Mode() os.FileMode  { return 0444 }
func (e *embeddedFileInfo) ModTime() time.Time { return e.time }
func (e *embeddedFileInfo) IsDir() bool        { return false }
func (e *embeddedFileInfo) Sys() interface{}   { return nil }

//for the root, needs to act as a directory...
type rootDirInfo time.Time

func (e rootDirInfo) Name() string       { return "/" }
func (e rootDirInfo) Size() int64        { return 0 }
func (e rootDirInfo) Mode() os.FileMode  { return 0555 }
func (e rootDirInfo) ModTime() time.Time { return time.Time(e) }
func (e rootDirInfo) IsDir() bool        { return true }
func (e rootDirInfo) Sys() interface{}   { return nil }
