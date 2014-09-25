package video

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"time"

	//our packages
	"code.7r.pm/chris/opfs/core"
	"code.7r.pm/chris/opfs/types/photo" //for the Orientation

	//vendor
	"github.com/disintegration/imaging"
)

func init() {
	//register the type on init
	factory := func() core.Inspecter {
		return &Mp4Video{}
	}
	//my mp4 code should work fine with .m4v and .mov types
	core.RegisterType("video/mp4", factory)
	core.RegisterType("video/x-m4v", factory)
	core.RegisterType("video/quicktime", factory)
}

//Our Mp4Video specific properties
type Mp4Video struct {
	Size        int64
	Width       int
	Height      int
	Orientation photo.ExifOrientation
	Duration    int64
}

func (p *Mp4Video) EnsureMeta(i *core.Item) error {
	if p.Size == 0 {
		//only do it if we have to
		return json.Unmarshal(i.Meta, p)
	}
	return nil
}

//This function is going to be the core of my scanner.
func (v *Mp4Video) Inspect(f io.ReadSeeker) (item *core.Item, err error) {
	item = &core.Item{
		Type: core.ItemTypeVideo,
	}

	//get the mvhd atom offset
	mvhd, err := getMp4MvhdAtom(f)
	if err != nil {
		return nil, fmt.Errorf("cannot find mvhd: %s", err)
	}

	//see if we can find the creation time
	created, err := getMp4CreationTimeFromMvhdAtom(f)
	if err != nil {
		return nil, fmt.Errorf("cannot get creation time: %s", err) //no creation, no dice
	}
	item.Created = core.AdjustTime(created)

	//return to mvhd to get duration
	f.Seek(mvhd, 0)
	duration, err := getMp4DurationFromMvhdAtom(f)
	if err == nil {
		v.Duration = duration
	}

	//getting size means finding the video track detail atom
	x, y, r, err := getMp4Dimensions(f)
	if err == nil {
		v.Width = int(x)
		v.Height = int(y)
		v.Orientation = r
	}

	latlon, err := getMp4Location(f)
	if err == nil {
		item.Location = latlon
	}

	//get size by seeking to end.
	v.Size, _ = f.Seek(0, 2)

	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	//set the metadata
	item.Meta = json.RawMessage(b)

	return
}

//try and find location from the "@xyz" "udta" in the "moov" atom.
func getMp4Location(f io.ReadSeeker) (ll *core.LatLon, err error) {
	f.Seek(0, 0) //rewind
	var head, size int64
	_, head, err = mp4FindAtom(f, "moov")
	if err != nil {
		//log.Println("cannot find moov")
		return
	}
	f.Seek(head, 1) //into moov atom.
	_, head, err = mp4FindAtom(f, "udta")
	if err != nil {
		//no udta atom!
		log.Println("cannot find udta")
		return
	}
	f.Seek(head, 1) //into udta atom
	size, head, err = mp4FindAtom(f, "\xa9xyz")
	if err != nil {
		//no ©xyz
		log.Println("cannot find \xa9xyz")
		return
	}
	//found it! the rest of the data is the string.
	f.Seek(head, 1) //into atom
	buff := make([]byte, size)
	if _, err = f.Read(buff); err != nil {
		return
	}
	//now to parse ISO6709 -> latlon!
	ll, err = parseISO6709(buff)
	return
}

//find the "mvhd" atom in an mp4 file.
func getMp4MvhdAtom(f io.ReadSeeker) (offset int64, err error) {
	f.Seek(0, 0) //rewind
	_, head, err := mp4FindAtom(f, "moov")
	if err != nil {
		//log.Println("cannot find moov")
		return
	}
	//seek into moov atom
	f.Seek(head, 1)
	if _, head, err = mp4FindAtom(f, "mvhd"); err != nil {
		//log.Println("cannot find mvhd")
		return
	}
	//seek into mvhd atom
	f.Seek(head, 1)
	offset, err = f.Seek(0, 1) //zero relative gives absolute offset
	return
}

//find the dimensions (and rotation!) of the mp4 video track
// @TODO: this gets the stored dimensions, but mp4
//        video can have a transformation matrix applied
//        meaning pixel resolution of source data != output video dimensions.
//        e.g. there may be a 90 rotation, or rectanglar pixels
func getMp4Dimensions(f io.ReadSeeker) (x, y int, r photo.ExifOrientation, err error) {
	//rewind
	f.Seek(0, 0)
	//find moov
	var head int64
	if _, head, err = mp4FindAtom(f, "moov"); err != nil {
		return
	}
	//seek into moov atom
	f.Seek(head, 1)
	//thre could be many trak headers
	var l, current, trakEnd int64
	buff := make([]byte, 4, 4)
	for {
		if trakEnd > 0 {
			f.Seek(trakEnd, 0)
			trakEnd = 0
		}
		//find trak
		if l, head, err = mp4FindAtom(f, "trak"); err != nil {
			return
		}
		current, _ = f.Seek(0, 1)
		trakEnd = current + l

		//now seek into it
		f.Seek(head, 1)
		//then media
		if _, head, err = mp4FindAtom(f, "tkhd"); err != nil {
			continue
		}

		f.Seek(head, 1)
		f.Read(buff)
		//this gives us version byte and flags.
		seek := int64(36) //initial seek to just before the "MatrixStructure"
		if buff[0] == 1 {
			//three fields are 64bit not 32. add 4*3
			seek += 4 * 3
		}
		f.Seek(seek, 1)
		//we want the first two fields of the matrix, which are cos(theta) and sin(theta)
		//of the rotation angle.
		sinTheta, e1 := getFixed32(f)
		cosTheta, e2 := getFixed32(f)

		if e1 != nil || e2 != nil {
			return
		}

		rotation := math.Atan2(cosTheta, sinTheta)

		log.Println("Rotation > cos:", cosTheta, "sin:", sinTheta, "rotation:", rotation*180/math.Pi)

		//now fix this to the nearest π/2 and map to Orientation
		//also set a flag to switch the x,y if a 90 type rotation.
		shouldSwitchDims := false
		switch int(180.0 * rotation / math.Pi) {
		case 0:
			//normal
			r = photo.OrientedNormal
		case 90, -270:
			//90 degrees.
			r = photo.OrientedNormal90
			shouldSwitchDims = true
		case 180, -180:
			//180
			r = photo.OrientedNormal180
		case 270, -90:
			//270
			r = photo.OrientedNormal270
			shouldSwitchDims = true
		}
		//now skip forward to the end of this matrix (7 more 4byte fields)
		f.Seek(7*4, 1)

		//now the dimensions:
		xf, e1 := getFixed32(f)
		yf, e2 := getFixed32(f)
		if e1 != nil || e2 != nil {
			return
		}

		if xf > 0 && yf > 0 {
			if shouldSwitchDims {
				x, y = int(yf), int(xf)
			} else {
				x, y = int(xf), int(yf)
			}
			break
		}
		//seek to end of trak, to try again
	}
	return
}

//read a 16.16 fixed 32bit float
func getFixed32(r io.ReadSeeker) (float64, error) {
	var s32 int32
	if err := binary.Read(r, binary.BigEndian, &s32); err != nil {
		return 0, err
	}
	return float64(s32) / float64(65536), nil
}

//pull the duration from an "mvhd" atom
func getMp4DurationFromMvhdAtom(f io.ReadSeeker) (duration int64, err error) {
	f.Seek(12, 1) //timescale is 12 bytes on.
	buff := make([]byte, 4, 4)
	_, err = f.Read(buff)
	if err != nil {
		return 0, err
	}
	timescale := binary.BigEndian.Uint32(buff)
	if timescale == 0 {
		return 0, fmt.Errorf("invalid timescale in mp4")
	}
	_, err = f.Read(buff)
	if err != nil {
		return 0, err
	}
	durationRaw := binary.BigEndian.Uint32(buff)
	//real duration is raw/timescale
	duration = int64((float64(durationRaw) / float64(timescale)) + 0.5)
	return
}

//now that is seconds since 1904-01-01 00:00:00
var mp4DateEpoch = core.TimeMustParse(time.RFC3339, "1904-01-01T00:00:00Z")

//to find the creation date from the "mvhd" atom, take bytes 5-8 as
//a Big Endian uint32, as the number of seconds since
// 1904-01-01 00:00:00 (?)
func getMp4CreationTimeFromMvhdAtom(f io.ReadSeeker) (time.Time, error) {
	//now we have it! bytes  5-8
	f.Seek(4, 1)
	buff := make([]byte, 4, 4)
	_, err := f.Read(buff)
	if err != nil {
		return time.Time{}, err
	}
	createdUint := binary.BigEndian.Uint32(buff)
	//and add our seconds
	return mp4DateEpoch.Add(time.Second * time.Duration(createdUint)), nil
}

//finds a "top level" atom from the current offset.
func mp4FindAtom(f io.ReadSeeker, atom string) (length, headerSize int64, err error) {
	buff := make([]byte, 4, 4)
	for {
		_, err = f.Read(buff)
		if err != nil {
			return 0, 0, err
		}
		l := int64(binary.BigEndian.Uint32(buff))

		_, err = f.Read(buff)
		if err != nil {
			return 0, 0, err
		}

		//log.Printf("Seeking Atom: `%s` found atom `%s` len: %d", atom, string(buff), l)
		//if the atom length is a "special value", then we need to do something else.
		//there are two "sepcial values"

		//l == 0 measn the rest of the file. so if we haven't found what we are looking for
		// we are done.
		// 1 means there is an extended length field 64-bits after the name
		if l == 1 {
			buff64 := make([]byte, 8, 8)
			l = int64(binary.BigEndian.Uint64(buff64)) //we already read 16 bytes
			headerSize = 16
		} else {
			headerSize = 8
		}

		f.Seek(-1*headerSize, 1)

		if string(buff) == atom {
			return l, headerSize, nil
		}
		if l == 0 {
			return 0, 0, fmt.Errorf("atom not found: %s", atom)
		}
		//we are at the start of the atom,
		//seek to the next.
		_, err = f.Seek(l, 1)
		if err != nil {
			return 0, 0, err
		}
	}
}

var (
	common6709 = regexp.MustCompile(`([-+]\d{2}(?:\.\d+)?)([-+]\d{3}(?:\.\d+)?)`)
)

func parseISO6709(b []byte) (*core.LatLon, error) {
	m := common6709.FindAllSubmatch(b, 2)
	if m == nil {
		log.Println("did not match iso6709:", string(b))
		return nil, fmt.Errorf("Error parsing ISO6709 format location")
	}
	var err error
	lat, lon := m[0][1], m[0][2]
	//these are strings though, lets convert.
	ll := &core.LatLon{}
	ll.Lat, err = strconv.ParseFloat(string(lat), 64)
	ll.Lon, err = strconv.ParseFloat(string(lon), 64)
	return ll, err
}

//Creating a thumbnail from a mp4 is complex so I cheat and use FFMPEG to create a JPEG...
//JPEG is straightforwards
// but ffmpeg probably can't make a thumbnail from a piped reader, so this only works if our
//ReadSeeker is actually an *os.File
func (m *Mp4Video) Thumbnail(in io.ReadSeeker, longSide int) (io.ReadSeeker, string, error) {
	var cmd *exec.Cmd
	if file, ok := in.(*os.File); ok {
		//this is the best way as ffmpeg can seek.
		cmd = exec.Command("ffmpeg", "-i", "/dev/fd/3", "-vframes", "1", "-f", "image2", "-")
		cmd.ExtraFiles = []*os.File{file}
	} else {
		log.Println("mp4thumb: using stdin (will probably fail...)")
		cmd = exec.Command("ffmpeg", "-i", "-", "-vframes", "1", "-f", "image2", "-")
		cmd.Stdin = in
	}
	stdout, err := cmd.StdoutPipe()
	//cmd.Stderr = os.Stderr
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	img, err := jpeg.Decode(stdout)
	if err != nil {
		return nil, "", err
	}
	if err := cmd.Wait(); err != nil {
		return nil, "", err
	}
	//now we should have a jpeg to resize!
	var w, h int
	aspect := float64(m.Width) / float64(m.Height)
	if m.Width > m.Height {
		w, h = longSide, int(float64(longSide)/aspect)
	} else {
		w, h = int(float64(longSide)*aspect), longSide
	}
	switch m.Orientation {
	case photo.OrientedNormal90, photo.OrientedNormal270:
		//flip then rotate 270
		w, h = h, w
	}
	//now create thumbnail.
	img = imaging.Thumbnail(img, w, h, imaging.Box)
	//rotate if needed.
	switch m.Orientation {
	case photo.OrientedNormal90:
		//rotate 90 (270 anticlockwise)
		img = imaging.Rotate270(img)
	case photo.OrientedNormal180:
		//rotate 180
		img = imaging.Rotate180(img)
	case photo.OrientedNormal270:
		//rotate 270 (90 anti-clockwise)
		img = imaging.Rotate90(img)
	}
	var wr bytes.Buffer
	err = jpeg.Encode(&wr, img, nil)
	return bytes.NewReader(wr.Bytes()), "image/jpeg", err
}
