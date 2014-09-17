package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	//our packages
	"code.7r.pm/chris/opfs/core"
	//these for side-effects
	_ "code.7r.pm/chris/opfs/types/photo"
	_ "code.7r.pm/chris/opfs/types/tag"
	_ "code.7r.pm/chris/opfs/types/video"
)

var (
	//default config location
	defaultConfigLocation = filepath.Join("~", ".opfs", "config.json")
	//configuration file
	config = flag.String("config", defaultConfigLocation, "config file location")
	//dump default config and exit
	dumpConfig = flag.Bool("dump-default-config", false, "output default config to stdout")
	//normal usage as API server
	daemon = flag.Bool("daemon", false, "Run API, Watch Import Dir and (maybe later) Serve UI")
	//single import
	importDir         = flag.String("import-dir", "", "Run Single Import Job on Directory")
	importTags        = flag.String("import-tags", "", "Tag for Single Import Jobs (can be more than one - comma-seperated)")
	importDeleteAfter = flag.Bool("import-delete", false, "Whether to delete successfully imported files")
	//re-index from store
	storeReindex = flag.Bool("store-reindex", false, "Reindex from the store")
	//debug flag, might be used for something...
	debug = flag.Bool("debug", false, "Show extra debug info (discloses error on the API)")
	//dump version and exit
	version = flag.Bool("version", false, "Show version and exit")
	//usage
	usage = flag.Bool("usage", false, "Print this usage text and exit")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Printf("OPFS %s\n", core.VERSION)
		return
	}
	if *usage {
		flag.Usage()
		return
	}
	if *dumpConfig {
		if err := core.WriteDefaultConfigTo(os.Stdout); err != nil {
			log.Fatal("Dump Config Error:", err)
		}
		os.Exit(0)
	}

	//make sure we use all available cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	conf, err := os.Open(core.ExpandHome(*config))
	configuration := &core.Config{}
	if err != nil {
		if os.IsNotExist(err) && *config == defaultConfigLocation {
			//this ok, probably first run. write config file.
			if err = core.WriteDefaultConfig(core.ExpandHome(*config)); err != nil {
				log.Fatal("error creating default config file:", err)
			}
			configuration = core.DefaultConfig
		} else {
			log.Fatal("Error opening config file:", err)
		}
	} else {
		if err = json.NewDecoder(conf).Decode(configuration); err != nil {
			log.Fatal("Error reading Config File:", err)
		}
		conf.Close()
	}

	//now instatiate from config.
	service, err := core.NewService(configuration, get_ui())
	if err != nil {
		log.Fatal("Error initialising service:", err)
	}

	if *importDir != "" {
		//single import job
		//work out tags

		var tags []string
		if *importTags != "" {
			tags = strings.Split(*importTags, ",")
		}

		opts := &core.ImportOptions{
			Dir:               *importDir,
			Tags:              tags,
			Time:              core.AdjustTime(time.Now()),
			MimeTypes:         core.AllMimeTypes(),
			DeleteAfterImport: *importDeleteAfter,
		}

		errors := service.Import(opts)
		for err := range errors {
			log.Println("ImportError:", err)
		}
		return
	}

	if *storeReindex {
		errors := service.ReindexFromStore()
		for err := range errors {
			log.Println("ImportError:", err)
		}
		return
	}

	if *daemon {
		log.Printf("OPFS: Starting API on  http://%s/\n", configuration.Api.Listen)
		var wg sync.WaitGroup
		wg.Add(1)
		errs, fini := service.WatchImport(&core.ImportOptions{
			MimeTypes:         core.AllMimeTypes(),
			DeleteAfterImport: true, //DANGER DANGER!
		})
		go func() {
			log.Println("API Fail:", service.ApiListen())
			wg.Done()
		}()
		go func() {
			for {
				select {
				case err := <-errs:
					log.Println("Import Watch Error:", err)
				case <-fini:
					break
				}
			}
		}()
		wg.Wait()
		//now shutdown
		close(fini)
		os.Exit(0)
	}

	//no option given...
	flag.Usage()
}
