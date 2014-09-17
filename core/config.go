package core

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	Store     *ServiceConfig //fs-store:./store
	Index     *ServiceConfig //es-index:http://127.0.0.1:9200/opfs
	Import    *ServiceConfig //fs-import:
	Export    *ServiceConfig //symlink-export:./export
	Api       *apiConfig     //127.0.0.1:4000
	CachePath string         //cache dir
}

type ServiceConfig struct {
	Type string // the type name registered by the store (e.g. fs-store)
	Conf map[string]interface{}
}

var DefaultConfig = &Config{
	Store: &ServiceConfig{
		Type: "fs-store",
		Conf: map[string]interface{}{},
	},
	Index: &ServiceConfig{
		Type: "es-indexer",
		Conf: map[string]interface{}{},
	},
	Import: &ServiceConfig{
		Type: "fs-import",
		Conf: map[string]interface{}{},
	},
	Export: &ServiceConfig{
		Type: "null-export",
		Conf: map[string]interface{}{},
	},
	Api: &apiConfig{
		Listen:     "127.0.0.1:4000",
		ShowErrors: true,
	},
	CachePath: filepath.Join("~", ".opfs", "cache"),
}

type apiConfig struct {
	Listen     string
	ShowErrors bool
}

//this is general. it's up to each factory to validate the
//given config
func ValidateConfig(conf *Config) {
	if conf.Store == nil {
		conf.Store = DefaultConfig.Store
	}
	if conf.Index == nil {
		conf.Index = DefaultConfig.Index
	}
	if conf.Import == nil {
		conf.Import = DefaultConfig.Import
	}
	if conf.Export == nil {
		conf.Export = DefaultConfig.Export
	}
	if conf.Api == nil {
		conf.Api = DefaultConfig.Api
	}
	if conf.CachePath == "" {
		conf.CachePath = DefaultConfig.CachePath
	}
}

func ExpandHome(s string) string {
	if s[0] == '~' {
		return os.Getenv("HOME") + s[1:len(s)]
	}
	return s
}

func WriteDefaultConfig(file string) error {
	err := os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return err
	}
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteDefaultConfigTo(f)
}

func WriteDefaultConfigTo(w io.Writer) error {
	b, err := json.MarshalIndent(DefaultConfig, "", "\t")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}
