// +build !embedui

package ui

import "net/http"

const IsEmbeddedUI = false

const DEFAULT_UI_PATH = "ui/www/"

func Handler(path *string) http.Handler {
	dir := *path
	if dir == "" {
		dir = DEFAULT_UI_PATH
	}
	return http.FileServer(&UI{fs: http.Dir(dir)})
}
