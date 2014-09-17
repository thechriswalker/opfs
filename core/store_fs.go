package core

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
)

const (
	METAFILE_EXT  = ".meta"
	METAFILE_MIME = "application/vnd.opfs.meta+json"
)

var (
	fsStoreDEFAULT_PATH = path.Join("~", ".opfs", "store")
)

func init() {
	mime.AddExtensionType(METAFILE_EXT, METAFILE_MIME)

	RegisterStore("fs-store", func(c *ServiceConfig, s *Service) (Store, error) {
		store, ok := c.Conf["path"]
		if !ok {
			store = fsStoreDEFAULT_PATH
		}
		fs_store, ok := store.(string)
		if !ok {
			return nil, fmt.Errorf("FS-Store Config `path` must be a string value: got `%v` (%T)", store, store)
		}
		return FileSystemStore(ExpandHome(fs_store)), nil
	})
}

type ErrDuplicateItem string

func (e ErrDuplicateItem) Error() string {
	return fmt.Sprintf("Duplicate Item: %s", string(e))
}

//file store keeps a base path and writes files into it.
//hash type is the bit before the first dash
// e.g. a sha1 hash looks like "sha1-70a0f0aeec2b80ca7dcb85e06781e8606ac8fee0"
// so hash type is "sha1" then the raw hash is after.
//the convention is to keep the file at <base>/<raw-hash[0:2]>/<raw-hash[2:4]>/<hash>
// and its associated metadata at the same location but ".meta"
//should be an absolute path.
type FileSystemStore string

var _ Store = FileSystemStore("")

//our paths are "/ab/cd/sha1-abcd..."
func getPath(hash string) string {
	return path.Join(hash[5:7], hash[7:9])
}

//retrieve the metadata for an item by hash
func (fs FileSystemStore) Meta(hash string) (*Item, error) {
	metafile := fs.Location(hash) + METAFILE_EXT
	return readMetafile(metafile)
}

//used by the exporter to find the file to export.
func (fs FileSystemStore) Location(hash string) string {
	return path.Join(string(fs), getPath(hash), hash)
}

func (fs FileSystemStore) Set(item *Item, rd io.Reader) (err error) {
	dir := path.Join(string(fs), getPath(item.Hash))
	name := path.Join(dir, item.Hash)
	//check if this exists. if it does, then we don't need to do anything.
	//or maybe we should merge the metadata. Currently I'm not going to
	//overwrite (it may clear tags we have set previously)
	if _, err = os.Stat(name); err == nil {
		//Already exists. Return Err Duplicate.
		//or do we just update the Meta?
		return ErrDuplicateItem(item.Hash)
	}

	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}
	//write file and meta
	res := make(chan error)
	go func() {
		//write meta.
		res <- fs.Update(item)
	}()
	go func() {
		//write data.
		f, err := os.Create(name)
		if err != nil {
			res <- err
			return
		}
		defer f.Close()
		_, err = io.Copy(f, rd)
		res <- err
	}()
	err1 := <-res
	err2 := <-res
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return
}

func (fs FileSystemStore) Get(hash string) (ReadSeekCloser, error) {
	file := path.Join(string(fs), getPath(hash), hash)
	return os.Open(file)
}

//update will assume that the directory structure exists. it is an update after all
func (fs FileSystemStore) Update(item *Item) error {
	hash := item.Hash
	file := path.Join(string(fs), getPath(hash), hash+".meta")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	return enc.Encode(item)
}

func (fs FileSystemStore) Iterate() (chan *Item, chan error) {
	out := doRecursiveScan(string(fs), map[string]struct{}{METAFILE_MIME: struct{}{}})
	errs := make(chan error)
	itemChan := make(chan *Item, 1)
	go func() {
		for sc := range out {
			if item, err := readMetafile(sc.Path); err == nil {
				itemChan <- item
			} else {
				errs <- err
			}
		}
		close(errs)
		close(itemChan)
	}()
	return itemChan, errs
}

func readMetafile(fullpath string) (item *Item, err error) {
	f, err := os.Open(fullpath)
	if err != nil {
		return
	}
	defer f.Close()
	item = &Item{}
	return item, json.NewDecoder(f).Decode(item)
}
