// +build embedui

package main

import (
	"net/http"

	"code.7r.pm/chris/opfs/ui"
)

func get_ui() http.Handler {
	return ui.Handler(nil)
}
