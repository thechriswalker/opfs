package core

import (
	"log"
	"net/http"
)

//This is what will actually be instantiated to control opfs.
type Service struct {
	importer Importer
	store    Store
	indexer  Indexer
	exporter Exporter
	cache    FileCache
	api      *Api
}

func NewService(conf *Config, ui http.Handler) (*Service, error) {
	//fill in any blanks
	ValidateConfig(conf)

	service := &Service{
		cache: FileCache(ExpandHome(conf.CachePath)),
	}
	var err error
	if service.importer, err = importerRegistryGet(conf.Import, service); err != nil {
		return nil, err
	}
	if service.store, err = storeRegistryGet(conf.Store, service); err != nil {
		return nil, err
	}
	if service.indexer, err = indexerRegistryGet(conf.Index, service); err != nil {
		return nil, err
	}
	if service.exporter, err = exporterRegistryGet(conf.Export, service); err != nil {
		return nil, err
	}
	service.api = &Api{
		service: service,
		listen:  conf.Api.Listen,
		headers: map[string]string{
			"X-Powered-By":   "OPFS",
			"X-OPFS-Version": VERSION,
		},
		ui:         ui,
		showErrors: conf.Api.ShowErrors,
	}
	service.api.initRoutes()

	return service, nil
}

func (s *Service) Version() string {
	return VERSION
}

//watch a directory for new items to import
func (s *Service) WatchImport(opts *ImportOptions) (chan error, chan struct{}) {
	shutdown := make(chan struct{})
	//catch kill signal and shutdown...
	return s.importer.Watch(opts, shutdown), shutdown
}

//Do a single import
func (s *Service) Import(opts *ImportOptions) chan error {
	return s.importer.Import(opts)
}

//Do a re-index form the data in the store.
func (s *Service) ReindexFromStore() chan error {
	errors := make(chan error)
	go func() {
		log.Println("Scanning Store and indexing")
		errs := ReindexStore(s.store, s.indexer)
		for e := range errs {
			errors <- e
		}
		// log.Println("Exporting Tags and Albums")
		// errs = ExportTags(s.exporter, s.indexer, s.store)
		// for e := range errs {
		// 	errors <- e
		// }
		// log.Println("Exporting Recents")
		// if err := ExportRecents(s.exporter, s.indexer, s.store); err != nil {
		// 	errors <- err
		// }
		close(errors)
	}()
	return errors
}

//start the API on a port
func (s *Service) ApiListen() error {
	return http.ListenAndServe(s.api.listen, s.api)
}
