package core

import (
	"encoding/json"
	"fmt"
)

var indexerRegistry = map[string]func(conf *ServiceConfig, s *Service) (Indexer, error){}

func RegisterIndexer(name string, factory func(conf *ServiceConfig, s *Service) (Indexer, error)) {
	indexerRegistry[name] = factory
}

func indexerRegistryGet(conf *ServiceConfig, s *Service) (Indexer, error) {
	factory, ok := indexerRegistry[conf.Type]
	if !ok {
		return nil, fmt.Errorf("Indexer does not exist: `%s`", conf.Type)
	}
	return factory(conf, s)
}

type Indexer interface {
	Index(item *Item) error                                      //index an item for searching,
	Search(query Query, page *Pagination) (*SearchResult, error) //perform a query. NB the Query should the same underlying type as returned from NewQuery
	NewQuery() Query                                             //get the start of a search query for this Index
	ListTags() ([]string, error)                                 //get all tags (list the available tags)
}

type Query interface {
	Type(t ...ItemType) Query                       //limit to one/more item type
	Match(field string, val ...interface{}) Query   //field match
	Range(field string, min, max interface{}) Query //range match, use nil for min/max for greater/less than
	Near(pos *LatLon, radius_in_km int) Query       //geos patial query
	Tagged(tag ...string) Query                     //with tags...
	Sort(field string, dir SortDir) Query           //set sort order
	AllowDeleted(allow bool) Query                  //whether to include deleted results, default = false
}

//used to specify sort direction
type SortDir bool

func (s *SortDir) UnmarshalText(b []byte) error {
	switch string(b) {
	case "asc":
		*s = SortDirAscending
	case "desc":
		*s = SortDirDescending
	default:
		return fmt.Errorf("Unknown sort direction: %s", string(b))
	}
	return nil
}

const (
	SortDirAscending  SortDir = true
	SortDirDescending SortDir = false
)

//offset and limit in our search query
type Pagination struct {
	From, Size int
}

//Result Holder. Directly JSON-able
type SearchResult struct {
	Count   int               //total number of results
	Page    *Pagination       //current pagination
	Next    *Pagination       //a pagination struct with details of how to make the next request. nil if there is no more
	Results []json.RawMessage //slice of the results as raw json bytes
}

//this is used as a placeholder to ensure non nil values
//we make it explicitly empty, so the first add will force
//an allocation (and dereference)
var emptySlice = make([]string, 0, 0)

func ReindexStore(s Store, i Indexer) chan error {
	items, errs := s.Iterate()
	errs2 := make(chan error)
	go func() {
		for item := range items {
			if err := i.Index(item); err != nil {
				errs2 <- err
			}
		}
		close(errs2)
	}()
	go func() {
		for err := range errs {
			errs2 <- err
		}
	}()
	return errs2
}
