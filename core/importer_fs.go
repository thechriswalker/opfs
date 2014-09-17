package core

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	//vendor
	"github.com/go-fsnotify/fsnotify"
)

var fsimpDEFAULT_WATCH_DIR = path.Join("~", "opfs-import")

func init() {
	RegisterImporter("fs-import", func(conf *ServiceConfig, s *Service) (Importer, error) {
		ipath, ok := conf.Conf["watch"]
		if !ok {
			ipath = ExpandHome(fsimpDEFAULT_WATCH_DIR)
		}
		fs_import, ok := ipath.(string)
		if !ok {
			return nil, fmt.Errorf("FS-Importer Config `watch` must be a string value: got `%v` (%T)", ipath, ipath)
		}
		imp := &FileSystemImporter{WatchDir: fs_import, service: s, pendingImports: map[string]func(){}, work: make(chan *fsImportJob)}

		//fire up the workers.
		imp.workers = make([]*fsImportWorker, runtime.GOMAXPROCS(-1))
		for i := range imp.workers {
			imp.workers[i] = newFsImportWorker(imp)
		}
		return imp, nil
	})
}

type FileSystemImporter struct {
	WatchDir       string
	pendingImports map[string]func()
	pendingLock    sync.Mutex
	service        *Service
	work           chan *fsImportJob
	workers        []*fsImportWorker
}

type fsImportJob struct {
	Result *ScanResult
	Opts   *ImportOptions
	Errors chan error
	Done   chan struct{}
}

type fsImportWorker struct {
	fs *FileSystemImporter
}

func newFsImportWorker(fs *FileSystemImporter) *fsImportWorker {
	w := &fsImportWorker{fs: fs}
	go func() {
		for j := range fs.work {
			if err := fs.importSingle(j.Result, j.Opts); err != nil {
				j.Errors <- err
			}
			j.Done <- struct{}{} //signal done
		}
	}()
	return w
}

var _ Importer = (*FileSystemImporter)(nil)

func (fs *FileSystemImporter) Import(opts *ImportOptions) chan error {
	errs := make(chan error)
	out := performScan(opts)
	done := make(chan struct{})
	var wg sync.WaitGroup
	go func() {
		for _ = range done {
			wg.Done()
		}
	}()
	go func() {
		for scan := range out {
			wg.Add(1)
			fs.work <- &fsImportJob{
				Result: scan,
				Opts:   opts,
				Errors: errs,
				Done:   done,
			}
		}

		wg.Wait()
		// ExportRecents(exp, index, store)
		close(errs)
	}()
	return errs
}

func (fs *FileSystemImporter) Watch(opts *ImportOptions, shutdown chan struct{}) chan error {
	//we watch opts.Dir or if empty, string(fs), or if emtpy, nothing.
	dir := opts.Dir
	if dir == "" {
		dir = fs.WatchDir
	}
	out := make(chan error)
	fini := func() {
		//do nothing.
		<-shutdown
		close(out)
	}
	if dir == "" {
		go fini()
	} else {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			out <- err
			go fini()
		} else {
			go func() {
				watcher, err := fsnotify.NewWatcher()
				if err != nil {
					out <- err
					fini()
					return
				}
				defer watcher.Close()
				if err = watcher.Add(dir); err != nil {
					fini()
					return
				}
				var wg sync.WaitGroup
				for {
					select {
					case ev := <-watcher.Events:
						//log.Println("ImportWatch Event:", ev)
						//probably only need write here...
						if ev.Op == fsnotify.Create || ev.Op == fsnotify.Write {
							//new file created let's have it!
							fs.pendingImport(ev.Name, opts, out, &wg)
						}
					case err := <-watcher.Errors:
						out <- err
					case <-shutdown:
						wg.Wait() //wait for anything happening to happen
						close(out)
						return
					}
				}
			}()
		}
	}

	return out
}

func (fs *FileSystemImporter) pendingImport(name string, opts *ImportOptions, out chan error, wg *sync.WaitGroup) {
	//check mutex
	fs.pendingLock.Lock()
	defer fs.pendingLock.Unlock()
	fn, ok := fs.pendingImports[name]
	if !ok {
		wg.Add(1) //need to add to the wait group now!

		fn = Debounce(func() {
			fs.pendingLock.Lock()
			delete(fs.pendingImports, name)
			fs.pendingLock.Unlock()
			info, err := os.Stat(name)
			if err != nil {
				out <- err
				return
			}
			if res := newScanResult(name, sliceToMap(opts.MimeTypes...), info); res != nil {
				done := make(chan struct{})
				fs.work <- &fsImportJob{
					Result: res,
					Opts:   opts,
					Errors: out,
					Done:   done,
				}
				<-done
			}
			wg.Done() //finally remove from wait group
		}, 5*time.Second) //5 second debounce should be enough for the slowest of writes...
		fs.pendingImports[name] = fn
	}
	fn()
}

func (fs *FileSystemImporter) importSingle(scan *ScanResult, opts *ImportOptions) error {
	//process is now:
	// find mime type.
	// inspect, which gives us the item.
	// look up existing, merge tags
	// overwrite added if needed
	// if created.isZero() then use file mtime
	// set in store
	// index
	// see if exportable -> export

	rd, err := os.Open(scan.Path)
	if err != nil {
		return err
	}
	defer rd.Close()
	item, err := Inspect(NewInspectableFile(rd, scan.Mime, path.Base(scan.Path)))
	if err != nil {
		return err
	}

	//check for existing
	if existing, err := fs.service.store.Meta(item.Hash); err == nil {
		//merge!
		mergeItemData(item, existing)
		//we just need to update!
		err = fs.service.store.Update(item)
		if err != nil {
			return err
		}
	} else {
		//new item
		rd.Seek(0, 0) //rewind
		err = fs.service.store.Set(item, rd)
		if err != nil {
			return err
		}
	}
	//(re)index
	if err = fs.service.indexer.Index(item); err != nil {
		return err
	}
	// if err = fs.service.exporter.ExportItem(item, store); err != nil {
	//  errs <- err
	//  //we don't need to continue here. We rebuild the index occasionally,
	//  //so we'll pick it up.
	// }
	if opts.DeleteAfterImport {
		if err = os.Remove(scan.Path); err != nil {
			return err
		}
	}
	log.Printf("Import success: %s (%s)\n", item.Name, item.Type)
	return nil
}
