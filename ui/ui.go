package ui

import (
	"log"
	"net/http"
	"path"
)

//the UI is static html. but we probably want to bundle the files in production
//and load dynamically in dev.
type UI struct {
	fs http.FileSystem
}

//the ui uses the HTML5 History API to change uri's, so we must serve
//the index.html for any unknown file.
func (u *UI) Open(name string) (http.File, error) {
	//somehow we seem to end up with double slashes here...
	//but the fileserver does a bunch of cleanup...
	name = path.Clean(name)
	file, err := u.fs.Open(name)
	if err != nil {
		//some problem, serve index and pretend.
		log.Println("Cannot open", name, "serving /index.html. Error:", err)
		return u.fs.Open("/index.html")
	}
	return file, err
}

type rootDir struct{}
