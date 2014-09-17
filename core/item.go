package core

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"time"
)

type ItemType uint16 //more types than I can imagine...

const (
	ItemTypeUnknown ItemType = iota //the zero value should be unknown
	ItemTypePhoto
	ItemTypeVideo
	ItemTypeTag
)

var (
	ErrUnknownItemType = fmt.Errorf("Unknown ItemType")
	ErrUnknownMimeType = fmt.Errorf("Unknown Mime Type")
)

func (i ItemType) String() string {
	switch i {
	case ItemTypePhoto:
		return "Photo"
	case ItemTypeVideo:
		return "Video"
	case ItemTypeTag:
		return "Tag"
	default:
		return "<unknown>"
	}
}

func (i ItemType) MarshalText() ([]byte, error) {
	s := i.String()
	if s == "<unknown>" {
		return nil, ErrUnknownItemType
	}
	return []byte(s), nil
}

func (i *ItemType) UnmarshalText(b []byte) error {
	switch string(b) {
	case "Photo":
		*i = ItemTypePhoto
	case "Video":
		*i = ItemTypeVideo
	case "Tag":
		*i = ItemTypeTag
	default:
		return ErrUnknownItemType
	}
	return nil
}

var typeMap = map[string]func() Inspecter{}

func RegisterType(mimeType string, factory func() Inspecter) {
	typeMap[mimeType] = factory
}

func AllMimeTypes() []string {
	m := make([]string, len(typeMap))
	i := 0
	for k, _ := range typeMap {
		m[i] = k
		i++
	}
	return m
}

//simply wrapper here.
func InspecterFor(item *Item) Inspecter {
	if f, ok := typeMap[item.Mime]; ok {
		return f()
	}
	return nil
}

type Inspectable interface {
	ReadSeeker() io.ReadSeeker
	MimeType() string
	Name() string
}

type InspectableFile struct {
	file       *os.File
	mime, name string
}

func (i *InspectableFile) ReadSeeker() io.ReadSeeker {
	return i.file
}

func (i InspectableFile) MimeType() string {
	return i.mime
}
func (i InspectableFile) Name() string {
	return i.name
}

func NewInspectableFile(f *os.File, mime, name string) Inspectable {
	return &InspectableFile{file: f, mime: mime, name: name}
}

// inspect a reader combined with mime-type to produce an item.
func Inspect(i Inspectable) (*Item, error) {
	if inspecter, ok := typeMap[i.MimeType()]; ok {
		item, err := inspecter().Inspect(i.ReadSeeker()) //this will fill in the Created,Location,Type and Meta of the Item.
		if err == nil {
			//create the hash
			item.Hash, err = generateHash(i.ReadSeeker())
		}
		if err != nil {
			return nil, err
		}
		//set mime type
		item.Mime = i.MimeType()
		item.Name = i.Name()
		//check for creation time
		if item.Created.IsZero() {
			item.Created = AdjustTime(time.Now())
		}
		if item.Location == nil {
			item.Location = &LatLon{} //put in an empty one...
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}

		//added is now.
		item.Added = AdjustTime(time.Now())
		return item, nil
	}
	return nil, ErrUnknownMimeType
}

//create a sha1 hash
func generateHash(r io.ReadSeeker) (string, error) {
	r.Seek(0, 0)
	hash := sha1.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha1-%x", hash.Sum(nil)), nil
}

//ensure time is UTC and truncated to second precision
func AdjustTime(t time.Time) time.Time {
	return t.UTC().Truncate(time.Second)
}

//wrapper to ensure time parsing happens
func TimeMustParse(layout, when string) time.Time {
	if t, err := time.Parse(layout, when); err != nil {
		panic(err)
	} else {
		return t
	}
}

//this function is intended on a reindex when we already have existing data.
//we want to add any new meta-data we might have been able to get from our Inspect
//function, but we want to retain the user-added content. That it Description/Tags
//so let's merge. We can't import with description, so we can set that directly.
//tags we might have, so we merge.
func mergeItemData(newItem, prevItem *Item) {
	// Added, Created time.Time       //times
	// Location       *LatLon         //where it was captured
	// Description    string          //long form description
	// Tags           []string        //the tags array that are associated with this item

	//Set the Added Time to the original added time.
	newItem.Added = prevItem.Added

	//check created time
	if newItem.Created.IsZero() && !prevItem.Created.IsZero() {
		//we manually added the creation time. best keep it.
		newItem.Created = prevItem.Created
	}

	//check location
	if newItem.Location.IsZero() && !prevItem.Location.IsZero() {
		//we manually added the creation time. best keep it.
		newItem.Location = prevItem.Location
	}

	//keep the description
	newItem.Description = prevItem.Description
	if len(prevItem.Tags) > 0 {
		if len(newItem.Tags) > 0 {
			//merge...
			newItem.Tags = append(prevItem.Tags, newItem.Tags...)
		} else {
			//replace
			newItem.Tags = prevItem.Tags
		}
	}
	//keep the original filename as well!
	newItem.Name = prevItem.Name
}

//this does the index/store update.
//@TODO, probably need to do some export work here too.
func item_update(s *Service, item *Item) error {
	if err := s.store.Update(item); err != nil {
		return err
	}
	return s.indexer.Index(item)
}
