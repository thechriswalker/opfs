package core

import (
	"encoding/json"
	"io"
	"time"
)

//this is the main type we use for all stored data.
type Item struct {
	Type           ItemType        //the overall item type (video/image/etc...)
	Mime           string          //mime-type of the raw file
	Hash           string          //hash of the data
	Name           string          //the original filename (for export/download)
	Added, Created time.Time       //times
	Deleted        *time.Time      `json:",omitempty"` //deletion time. pointer so omitempty works...
	Location       *LatLon         //where it was captured
	Description    string          //long form description
	Tags           []string        //the tags array that are associated with this item
	Meta           json.RawMessage //raw JSON of the meta. We shouldn't need to worry about this.
}

//minimal interface is Inspect a reader and initialise from raw json
type Inspecter interface {
	//This is what allows us to import items from a source. The basic info is gathered
	//by the importer, type, hash, and then the created time and location are got from the file
	//also any extra meta is dumped here (e.g. image width/height/orientation, video duration)
	Inspect(io.ReadSeeker) (item *Item, err error)
	EnsureMeta(item *Item) error //this is the unmarshal process.
}

//Thumbnailer interface is something that has a thumbnail
type Thumbnailer interface {
	Thumbnail(in io.ReadSeeker, maxSide int) (rd io.ReadSeeker, mime string, err error)
}
