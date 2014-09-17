package core

import (
	"bytes"
	"fmt"
	"io"
)

//it takes too long to create thumbnails on the fly. So we need to make a queue,
//and process only one at a time. and we need to ensure we don't try to make more than
//one at once of the same item.
//also, maybe the thumbnail generator should be responsible for the caching, to ensure
//only on at a time...

//1x1 pixel transparent gif
var TINY_GIF = []byte{71, 73, 70, 56, 57, 97, 1, 0, 1, 0, 128, 0, 0, 255, 255, 255, 0, 0, 0, 44, 0, 0, 0, 0, 1, 0, 1, 0, 0, 2, 2, 68, 1, 0, 59}

type readSeekerMimeType struct {
	read io.ReadSeeker
	mime string
}

var thumbGroup = &DoGroup{}

func getItemThumbnail(s *Service, i *Item, size int) (io.ReadSeeker, string, error) {
	key := fmt.Sprintf("thumb-%s-%d", i.Hash, size)
	fn := func() (interface{}, error) {
		//first check cache.
		if cached, err := s.cache.Get(key); err == nil {
			return &readSeekerMimeType{read: cached}, nil
		}
		//no cache, create file.
		rsc, err := s.store.Get(i.Hash)
		if err != nil {
			return nil, err
		}
		defer rsc.Close()
		var r io.ReadSeeker
		var m string

		//to get the type factory that may be capable of creating a thumbnail we use
		//the typeMap
		f := InspecterFor(i)
		var ok bool
		var tmb Thumbnailer
		if f != nil {
			tmb, ok = f.(Thumbnailer)
		}
		if ok {
			if err = f.EnsureMeta(i); err == nil {
				r, m, err = tmb.Thumbnail(rsc, size)
			}
		} else {
			//fake it
			return &readSeekerMimeType{
				read: bytes.NewReader(createGif(uint16(size), uint16(size))),
				mime: "image/gif",
			}, nil
		}
		if err != nil {
			return nil, err
		}
		s.cache.SetReader(key, r)
		return &readSeekerMimeType{read: r, mime: m}, nil
	}
	r, err := thumbGroup.Do(key, fn)
	if err != nil {
		return nil, "", err
	}
	rsmt := r.(*readSeekerMimeType)
	return rsmt.read, rsmt.mime, nil
}
