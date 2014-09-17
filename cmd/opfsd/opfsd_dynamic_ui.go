// +build !embedui

package main

import (
	"flag"
	"net/http"

	"code.7r.pm/chris/opfs/ui"
)

var uiPath = flag.String("ui", ui.DEFAULT_UI_PATH, "path to UI directory to serve")

func get_ui() http.Handler {
	return ui.Handler(uiPath)
}
