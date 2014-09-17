package photo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"math/big"
	"time"

	//our packages
	"code.7r.pm/chris/opfs/core"

	//vendor
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

//register the type on init
func init() {
	core.RegisterType("image/jpeg", func() core.Inspecter {
		return &JpegPhoto{}
	})
}

//This is what dates look like in EXIF tags.
const ExifDateFormat = "2006:01:02 15:04:05"

//apparently some phones/cameras have a bug which sets some of the exif values
//to 2002-12-08 12:00:00 +0000 UTC.
//so we need to ensure we ignore that date.
var exifDateBug = core.TimeMustParse(time.RFC3339, "2002-12-08T12:00:00Z")

//these fields might contain exif date-time strings (in "most-likely-to-be-correct" order)
var possibleExifDateTimeFields = []exif.FieldName{exif.DateTimeDigitized, exif.DateTime, exif.DateTimeOriginal}

//exif orientation flag
type ExifOrientation uint

//we define the angles in degrees clockwise.
//normal means no transform before rotation
//mirror means flip along a vertical axis before
//rotation, Undefined should be treated as Normal
const (
	OrientedUndefined ExifOrientation = iota
	OrientedNormal                    //1
	OrientedMirror                    //2
	OrientedNormal180                 //3
	OrientedMirror180                 //4
	OrientedMirror270                 //5
	OrientedNormal270                 //6
	OrientedMirror90                  //7
	OrientedNormal90
)

//our jpeg photo metadata
type JpegPhoto struct {
	Size        int64
	Width       int
	Height      int
	Orientation ExifOrientation
	Device      string //exif make and model
}

func (p *JpegPhoto) EnsureMeta(i *core.Item) error {
	if p.Size == 0 {
		//only do it if we have to
		return json.Unmarshal(i.Meta, p)
	}
	return nil
}

//This function is going to be the core of my scanner.
func (p *JpegPhoto) Inspect(f io.ReadSeeker) (item *core.Item, err error) {
	item = &core.Item{
		Type:     core.ItemTypePhoto,
		Location: &core.LatLon{},
	}
	p.Width = 0
	p.Height = 0
	p.Orientation = OrientedNormal
	p.Device = ""

	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	p.Width = cfg.Width
	p.Height = cfg.Height

	f.Seek(0, 0) //rewind...
	//get date from exif
	exifInfo, err := exif.Decode(f)
	if err != nil {
		return nil, err //no exif, no photo
	}
	var createds []time.Time

	for _, field := range possibleExifDateTimeFields {
		tag, err := exifInfo.Get(field)
		if err != nil {
			//log.Println("no tag", field)
			continue
		}
		if tag.TypeCategory() != tiff.StringVal {
			log.Println("wrong type", field)
			continue
		}
		if tag_created, err := time.Parse(ExifDateFormat, tag.StringVal()); err == nil {
			if tag_created == exifDateBug {
				log.Println("EXIF DATE BUG:", field)
				continue
			}
			createds = append(createds, tag_created)
		}
	}
	if len(createds) == 0 || createds[0].IsZero() {
		return nil, fmt.Errorf("Could not get date photo taken")
	}

	item.Created = core.AdjustTime(createds[0])

	//now optional, orientation
	tag, err := exifInfo.Get(exif.Orientation)
	if err == nil {
		p.Orientation = ExifOrientation(tag.Int(0))
		//swap height/width if needed.
		switch p.Orientation {
		case OrientedMirror270, OrientedNormal270, OrientedNormal90, OrientedMirror90:
			p.Width, p.Height = p.Height, p.Width
		}
	}

	//Device/Make
	tag, err = exifInfo.Get(exif.Make)
	if err == nil && tag.TypeCategory() == tiff.StringVal {
		p.Device = tag.StringVal()
	}
	tag, err = exifInfo.Get(exif.Model)
	if err == nil && tag.TypeCategory() == tiff.StringVal {
		if p.Device == "" {
			p.Device = tag.StringVal()
		} else {
			p.Device = p.Device + " " + tag.StringVal()
		}
	}

	//and GPS location
	setLocationFromExif(item.Location, exifInfo)

	//get size of file by seeking to the end.
	p.Size, _ = f.Seek(0, 2)

	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	//now add meta to item.
	item.Meta = json.RawMessage(b)

	return
}

//best effort set location from exif data.
func setLocationFromExif(l *core.LatLon, x *exif.Exif) {
	lat, err := x.Get(exif.GPSLatitude)
	if err != nil || lat.TypeCategory() != tiff.RatVal || lat.Count != 3 {
		return
	}
	lon, err := x.Get(exif.GPSLongitude)
	if err != nil || lon.TypeCategory() != tiff.RatVal || lon.Count != 3 {
		return
	}
	//we should have 3 rational values. and this lib panics on fail.
	//so we did the checks first.

	l.Lat = toDecimalDegreesFromRat(lat.Rat(0), lat.Rat(1), lat.Rat(2))
	l.Lon = toDecimalDegreesFromRat(lon.Rat(0), lon.Rat(1), lon.Rat(2))
	//l is a pointer, so we have filled in the values now,
	// and do not need to return anything.
}

//60 in big.Rat format for conversion
var rational60 = big.NewRat(60, 1)

//Convert three big.Rat's in degrees, minutes and seconds to a float64 decimal degrees
func toDecimalDegreesFromRat(deg, min, sec *big.Rat) float64 {
	sec.Quo(sec, rational60) //divide seconds by 60
	min.Add(min, sec)        //add seconds to mins
	min.Quo(min, rational60) //divide mins by 60
	deg.Add(deg, min)        //add mins to degrees
	f, _ := deg.Float64()    //get approx float
	return f
}

//Creating a thumbnail from a JPEG is straightforwards
func (p *JpegPhoto) Thumbnail(in io.ReadSeeker, longSide int) (io.ReadSeeker, string, error) {

	//first we need to read the image.
	img, _, err := image.Decode(in)
	if err != nil {
		return nil, "", err
	}
	var w, h int
	aspect := float64(p.Width) / float64(p.Height)
	if p.Width > p.Height {
		w, h = longSide, int(float64(longSide)/aspect)
	} else {
		w, h = int(float64(longSide)*aspect), longSide
	}
	//we need to do this switch twice. first to check if we need to swap width/height
	//as we resize before rotation/flip
	//then after to do the resize/flip.
	switch p.Orientation {
	case OrientedNormal90, OrientedMirror90, OrientedNormal270, OrientedMirror270:
		//flip then rotate 270
		w, h = h, w
	}
	//now create thumbnail.
	img = imaging.Thumbnail(img, w, h, imaging.Box)
	//now we need to rotate/flip it to match the ExifOrientation flag
	switch p.Orientation {
	case OrientedNormal:
		//nothing
	case OrientedMirror:
		//flip only
		img = imaging.FlipH(img)
	case OrientedNormal90:
		//rotate 90
		img = imaging.Rotate90(img)
	case OrientedMirror90:
		//flip and rotate 90
		img = imaging.FlipH(imaging.Rotate90(img))
	case OrientedNormal180:
		//rotate 180
		img = imaging.Rotate180(img)
	case OrientedMirror180:
		//flip then rotate 180
		img = imaging.FlipH(imaging.Rotate180(img))
	case OrientedNormal270:
		//rotate 270 (90 anti-clockwise)
		img = imaging.Rotate270(img)
	case OrientedMirror270:
		//flip then rotate 270
		img = imaging.FlipH(imaging.Rotate270(img))
	}
	//now re-encode
	var wr bytes.Buffer
	err = jpeg.Encode(&wr, img, nil)
	return bytes.NewReader(wr.Bytes()), "image/jpeg", err
}
