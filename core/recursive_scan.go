package core

import (
	"io"
	"log"
	"mime"
	"os"
	"path"
	"strings"
)

const BATCH_SIZE = 100

type ScanResult struct {
	Path string
	Mime string
	Info os.FileInfo
}

func doRecursiveScan(dir string, mimeMap map[string]struct{}) chan *ScanResult {
	out := make(chan *ScanResult)
	go func() {
		recursiveScan(dir, mimeMap, out)
		close(out)
	}()
	return out
}

//Finds new entrys in the dir
func recursiveScan(dir string, mimeMap map[string]struct{}, out chan *ScanResult) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	defer d.Close()
	for {
		fi, err := d.Readdir(BATCH_SIZE)
		if err == io.EOF {
			return
		}
		if err != nil {
			panic(err)
		}
		for _, f := range fi {
			full := path.Join(dir, f.Name())
			switch {
			case f.IsDir():
				recursiveScan(full, mimeMap, out)
			default:
				if res := newScanResult(full, mimeMap, f); res != nil {
					out <- res
				}
			}
		}
	}
}

func newScanResult(fullpath string, mimeMap map[string]struct{}, info os.FileInfo) *ScanResult {
	m := mime.TypeByExtension(strings.ToLower(path.Ext(fullpath)))
	if _, ok := mimeMap[m]; ok {
		return &ScanResult{
			Path: fullpath,
			Mime: m,
			Info: info,
		}
	}
	log.Println(info.Name(), "Unwanted Mime Type:", m)
	return nil
}
