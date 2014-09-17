package core

import (
	"fmt"
	"time"
)

type Importer interface {
	Import(*ImportOptions) chan error
	Watch(*ImportOptions, chan struct{}) chan error
}

var importerRegistry = map[string]func(conf *ServiceConfig, s *Service) (Importer, error){}

func RegisterImporter(name string, factory func(conf *ServiceConfig, s *Service) (Importer, error)) {
	importerRegistry[name] = factory
}

func importerRegistryGet(conf *ServiceConfig, s *Service) (Importer, error) {
	factory, ok := importerRegistry[conf.Type]
	if !ok {
		return nil, fmt.Errorf("Importer does not exist: `%s`", conf.Type)
	}
	return factory(conf, s)
}

type ImportOptions struct {
	Dir               string    //directory to scan for photos.
	MimeTypes         []string  //which mime-types to import.
	Tags              []string  //tags to auto-tag the imports.
	Time              time.Time //time to set as "added". Ensures all imports in a batch have exactly the same added time
	DeleteAfterImport bool      //whether or not to auto-delete the source after import
}

func performScan(opts *ImportOptions) chan *ScanResult {
	mimeMap := sliceToMap(opts.MimeTypes...)
	return doRecursiveScan(opts.Dir, mimeMap)
}

func sliceToMap(s ...string) map[string]struct{} {
	smap := make(map[string]struct{}, len(s))
	for _, e := range s {
		smap[e] = struct{}{}
	}
	return smap
}
