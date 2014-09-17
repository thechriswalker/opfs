package core

import (
	"fmt"
	"io"
)

//io doesn't have this!
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

//note you cannot set data without meta, as any chagne to data would change the hash.
//so it is a new entry
type Store interface {
	Set(item *Item, r io.Reader) error       //set data with meta in the store
	Get(hash string) (ReadSeekCloser, error) //get data from the store, callers responsibility to close the reader
	Meta(hash string) (*Item, error)         //get meta
	Update(item *Item) error                 //update metadata in the store.
	Iterate() (chan *Item, chan error)       //get an iterator on this store
	Location(hash string) string             //returns the full path to this item (if available)
}

var storeRegistry = map[string]func(conf *ServiceConfig, s *Service) (Store, error){}

func RegisterStore(name string, factory func(conf *ServiceConfig, s *Service) (Store, error)) {
	storeRegistry[name] = factory
}

func storeRegistryGet(conf *ServiceConfig, s *Service) (Store, error) {
	factory, ok := storeRegistry[conf.Type]
	if !ok {
		return nil, fmt.Errorf("Store does not exist: `%s`", conf.Type)
	}
	return factory(conf, s)
}
