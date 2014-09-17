package core

import (
	"fmt"
)

var exporterRegistry = map[string]func(conf *ServiceConfig, s *Service) (Exporter, error){}

func RegisterExporter(name string, factory func(conf *ServiceConfig, s *Service) (Exporter, error)) {
	exporterRegistry[name] = factory
}

func exporterRegistryGet(conf *ServiceConfig, s *Service) (Exporter, error) {
	factory, ok := exporterRegistry[conf.Type]
	if !ok {
		return nil, fmt.Errorf("Exporter does not exist: `%s`", conf.Type)
	}
	return factory(conf, s)
}

type Exporter interface {
	//flatten the whole export path to start again.
	Flatten(dir string) error
	//export some media to a dir
	Export(dir string, items []*Item) error
	//export a single item, assuming this is just added, do tags and albums
	ExportItem(item *Item) error
}

// type SymlinkExporter string

// func (s SymlinkExporter) Export(dir string, items []*Item, store Store) error {
// 	var basedir = path.Join(string(s), dir)
// 	if err := os.MkdirAll(basedir, 0700); err != nil {
// 		return err
// 	}
// 	var from, to string
// 	for _, i := range items {
// 		if e, ok := i.GetExporter(); ok {
// 			from = store.Location(i.Hash)
// 			to = path.Join(basedir, e.FileName())
// 			if err := os.Symlink(from, to); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }

// func (s SymlinkExporter) ExportItem(item *Item, store Store) (err error) {
// 	//we need to export tags/albums/starred
// 	// tagset := item.GetTags()
// 	// for _, tag := range tagset.Regular {
// 	// 	err = s.Export(path.Join("tag", tag), []Item{item}, store)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// }
// 	// for _, tag := range tagset.Albums {
// 	// 	err = s.Export(path.Join("album", tag), []Item{item}, store)
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// }
// 	// if tagset.Starred {
// 	// 	err = s.Export("starred", []Item{item}, store)
// 	// }
// 	return
// }

// func (s SymlinkExporter) Flatten(dir string) error {
// 	return os.RemoveAll(path.Join(string(s), dir))
// }

// //tags covers albums/tags/etc..
// func ExportTags(ex Exporter, index Indexer, store Store) chan error {
// 	errs := make(chan error)
// 	close(errs)
// 	return errs

// 	// errs := make(chan error)
// 	// go func() {
// 	// 	tags, err := index.ListTags()
// 	// 	if err != nil {
// 	// 		errs <- err
// 	// 		close(errs)
// 	// 		return
// 	// 	}
// 	// 	for _, tag := range tags {
// 	// 		dir := path.Join(strings.SplitN(tag, TagPrefixDelimiter, 2)...)
// 	// 		if err := ex.Flatten(dir); err != nil {
// 	// 			errs <- err
// 	// 			continue
// 	// 		}
// 	// 		page := &Pagination{0, 100}
// 	// 		query := index.NewQuery().Tagged(tag)
// 	// 		itemSlice := make([]Item, 0, 100)
// 	// 		var item Item
// 	// 		for {
// 	// 			res, err := index.Search(query, page)
// 	// 			if err != nil {
// 	// 				errs <- err
// 	// 				break
// 	// 			}
// 	// 			itemSlice = itemSlice[0:0]
// 	// 			for _, hit := range res.Results {
// 	// 				switch hit.Type {
// 	// 				case ItemTypePhoto:
// 	// 					item, err = PhotoFromRaw(hit.JSON)
// 	// 				case ItemTypeVideo:
// 	// 					item, err = VideoFromRaw(hit.JSON)
// 	// 				default:
// 	// 					err = fmt.Errorf("Invalid Type Found: %s", hit.Type)
// 	// 				}
// 	// 				if err != nil {
// 	// 					errs <- err
// 	// 					continue
// 	// 				}
// 	// 				itemSlice = append(itemSlice, item)
// 	// 			}
// 	// 			err = ex.Export(dir, itemSlice, store)
// 	// 			if err != nil {
// 	// 				errs <- err
// 	// 			}
// 	// 			if res.Next == nil {
// 	// 				//we are done! (with this tag)
// 	// 				break
// 	// 			}
// 	// 			//on to the next.
// 	// 			page = res.Next
// 	// 		}
// 	// 	}
// 	// 	close(errs)
// 	// }()
// 	// return errs
// }

// func ExportRecents(ex Exporter, index Indexer, store Store) error {
// 	// //search for top X recent photos.
// 	// q := index.NewQuery().Sort("Created", SortDirDescending)
// 	// p := &Pagination{0, RECENT_EXPORTS_SIZE}
// 	// res, err := index.Search(q, p)
// 	// if err != nil {
// 	// 	return err
// 	// }
// 	// itemSlice := make([]Item, 0, len(res.Results))
// 	// var item Item
// 	// for _, hit := range res.Results {
// 	// 	switch hit.Type {
// 	// 	case ItemTypePhoto:
// 	// 		item, err = PhotoFromRaw(hit.JSON)
// 	// 	case ItemTypeVideo:
// 	// 		item, err = VideoFromRaw(hit.JSON)
// 	// 	default:
// 	// 		continue
// 	// 	}
// 	// 	if err != nil {
// 	// 		return err
// 	// 	}
// 	// 	itemSlice = append(itemSlice, item)
// 	// }
// 	// //flatten
// 	// if err := ex.Flatten(RECENT_DIR); err != nil {
// 	// 	return err
// 	// }
// 	// //export
// 	// return ex.Export(RECENT_DIR, itemSlice, store)
// 	return nil
// }
