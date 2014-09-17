package core

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
)

//our filecache is simply a cache currently only used for thumbnails.
//super simple.
type FileCache string

func (f FileCache) SetReader(key string, r io.Reader) error {
	dir := f.dir(key)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	c, err := os.Create(path.Join(dir, key))
	if err != nil {
		return err
	}
	_, err = io.Copy(c, r)
	return err
}

func (f FileCache) SetBytes(key string, b []byte) error {
	dir := f.dir(key)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	c, err := os.Create(path.Join(dir, key))
	if err != nil {
		return err
	}
	_, err = c.Write(b)
	return err
}

func (f FileCache) Get(key string) (rsc ReadSeekCloser, err error) {
	return os.Open(path.Join(f.dir(key), key))
}

func (f FileCache) dir(key string) string {
	//hash it fast and take the first 4-hex chars... 2/2
	hex := fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return path.Join(string(f), hex[0:2], hex[2:4])
}
