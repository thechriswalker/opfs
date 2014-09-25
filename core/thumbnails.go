package core

import (
	"bytes"
	"fmt"
	"io"

	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	//vendor
	"github.com/disintegration/imaging"
)

//it takes too long to create thumbnails on the fly. So we need to make a queue,
//and process only one at a time. and we need to ensure we don't try to make more than
//one at once of the same item.
//also, maybe the thumbnail generator should be responsible for the caching, to ensure
//only on at a time...

var CONCURRENT_THUMBNAIL_CREATE_LIMIT = 4

//1x1 pixel transparent gif
var TINY_GIF = []byte{71, 73, 70, 56, 57, 97, 1, 0, 1, 0, 128, 0, 0, 255, 255, 255, 0, 0, 0, 44, 0, 0, 0, 0, 1, 0, 1, 0, 0, 2, 2, 68, 1, 0, 59}

type readSeekerMimeType struct {
	read io.ReadSeeker
	mime string
}

var thumbLimit = make(chan struct{}, CONCURRENT_THUMBNAIL_CREATE_LIMIT)

var thumbGroup = &DoGroup{}

func getItemThumbnail(s *Service, i *Item, size int) (io.ReadSeeker, string, error) {
	key := fmt.Sprintf("thumb-%s-%d", i.Hash, size)
	fn := func() (interface{}, error) {

		//first check cache.
		if cached, err := s.cache.Get(key); err == nil {
			return &readSeekerMimeType{read: cached}, nil
		}

		//massive short cut here. if we have a "large" thumb then just resize that...
		if size < THUMBNAIL_LARGE {
			//check cache for LARGE
			key := fmt.Sprintf("thumb-%s-%d", i.Hash, THUMBNAIL_LARGE)
			if cached, err := s.cache.Get(key); err == nil {
				//resize this.
				if img, _, err := image.Decode(cached); err != nil {
					img = imaging.Fit(img, size, size, imaging.Box)
					var wr bytes.Buffer
					if err := jpeg.Encode(&wr, img, nil); err != nil {
						r := bytes.NewReader(wr.Bytes())
						s.cache.SetReader(key, r)
						return &readSeekerMimeType{read: r, mime: "image/jpeg"}, nil
					}
				}
			}
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
				//ensure we don't try to make too many at once...
				//before we block on putting a token into the bucket. if the bucket is full this will block.
				//after we take a token out of the bucket, so another can process.
				thumbLimit <- struct{}{}
				r, m, err = tmb.Thumbnail(rsc, size)
				<-thumbLimit
				s.cache.SetReader(key, r)
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
		return &readSeekerMimeType{read: r, mime: m}, nil
	}
	r, err := thumbGroup.Do(key, fn)
	if err != nil {
		return nil, "", err
	}
	rsmt := r.(*readSeekerMimeType)
	return rsmt.read, rsmt.mime, nil
}
