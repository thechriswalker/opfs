package main

//this is the build script.

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"text/template"
)

const (
	PKGBASE         = "code.7r.pm/chris/opfs/"
	UI_BASE         = ""
	UI_AUTOGEN_FILE = "ui/ui_autogenerated_resources.go"
	UI_SOURCE_DIR   = "ui/www"
	UI_TEMPLATE     = `// +build embedui

//
// WARNING! AUTO-GENERATED FILE
//    do not edit by hand
//
package ui

import (
	"bytes"
  "encoding/base64"
  "time"
)

var mustDecode = func(s string) (b []byte) {
  var e error
  if b, e = base64.StdEncoding.DecodeString(s); e != nil {
    panic(e)
  }
  return b
}

var embeddedUI = EmbeddedHttpFileSystem{
  "/": &embeddedFile{reader: nil, stat: rootDirInfo(time.Now())},{{range .}}
  "{{.Name}}": &embeddedFile{
    reader: bytes.NewReader(mustDecode("{{.Data}}")),
    stat:   &embeddedFileInfo{
      size: {{.Size}},
      name: "{{.Basename}}",
      time: time.Unix({{.ModTimeSec}}, 0),
    },
	},{{end}}
}
`
)

const binaryDefaultName = "bin/opfsd"

var tpl = template.Must(template.New("eui").Parse(UI_TEMPLATE))

type file struct {
	Name       string
	Size       int64
	ModTimeSec int64
	f          *os.File
}

func (f *file) Basename() string {
	return filepath.Base(f.Name)
}

//returns a base64 representation of the file...
func (f *file) Data() string {
	data, err := ioutil.ReadAll(f.f)
	if err != nil {
		log.Println(f.Name, "read error", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func euiChan(root string) chan *file {
	ch := make(chan *file)
	go func() {
		defer func() { close(ch) }()
		filepath.Walk(root, filepath.WalkFunc(func(name string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || filepath.Base(name)[0] == '.' {
				return nil //just keep going
			}
			pathname := name[len(root):]
			log.Println("Embedding File: ", pathname)
			f, err := os.Open(name)
			if err != nil {
				log.Println("Open Err: ", err)
				return err //barf!
			}
			if filepath.Separator != '/' {
				pathname = strings.Replace(pathname, string(filepath.Separator), "/", -1)
			}
			ch <- &file{
				Name:       pathname,
				Size:       info.Size(),
				ModTimeSec: info.ModTime().Unix(),
				f:          f,
			}
			//No Error
			return nil
		}))
	}()
	return ch
}

func createEmbeddedUIFile(src, dst string) error {
	wr, err := os.Create(dst)
	if err != nil {
		return err
	}
	data := euiChan(src)
	return tpl.Execute(wr, data)
}

var (
	version    = flag.String("version", "", "set the version manually")
	embedUI    = flag.Bool("embed-ui", true, "whether to embed the default UI")
	buildUI    = flag.Bool("build-ui", true, "whether run the UI build script before embedding")
	targetARCH = flag.String("arch", runtime.GOARCH, "Architecture to build for.")
	targetOS   = flag.String("os", runtime.GOOS, "Operating system to build for.")
)

func main() {
	flag.Parse()

	//platform/arch GOOS and GOARCH
	//	matchOS := *targetOS == runtime.GOOS
	//	matchARCH := *targetARCH == runtime.GOARCH

	var binaryName string
	//	if !matchOS || !matchARCH {
	binaryName = fmt.Sprintf("%s-%s-%s", binaryDefaultName, *targetOS, *targetARCH)
	//	} else {
	//		binaryName = binaryDefaultName
	//	}

	buildArgs := []string{
		"build", "-o", binaryName,
	}

	if *version == "" {
		v, err := getVersionFromGit()
		if err != nil {
			log.Fatalln("Could not get version automatically: ", err)
		}
		*version = v
	}

	log.Println("Build Version:", *version)
	buildArgs = append(buildArgs, "-ldflags", "-X "+PKGBASE+"core.VERSION "+*version)

	if *embedUI {
		if *buildUI {
			if err := doUIBuild(); err != nil {
				log.Fatalln("Failed to build UI", err)
			}
		}

		if err := createEmbeddedUIFile(UI_SOURCE_DIR, UI_AUTOGEN_FILE); err != nil {
			log.Fatal("Failed to generate embedded files:", err)
		}
		buildArgs = append(buildArgs, "--tags=embedui")
		defer os.Remove(UI_AUTOGEN_FILE)
	}

	buildArgs = append(buildArgs, "./cmd/opfsd")

	//now build

	env := environment()
	log.Println("ENV:", env)
	//ensure the core is up to date. not sure why go doesn't do this auto-magically...
	for _, pkg := range []string{"core", "types/video", "types/photo", "types/tag"} {
		cmd := exec.Command("go", "install", PKGBASE+pkg)
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalln(err)
		}
	}

	log.Println("Building opfsd binary...")
	log.Println("CMD: go", strings.Join(buildArgs, " "))

	cmd := exec.Command("go", buildArgs...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}
}

//get the environment for our build command.
//almost the same as our regular env, but with
// GOOS/GOARCH set
func environment() []string {
	env := []string{"GOOS=" + *targetOS, "GOARCH=" + *targetARCH}
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOOS=") || strings.HasPrefix(e, "GOARCH=") {
			//ignore
			continue
		}
		env = append(env, e)
	}
	return env
}

func getVersionFromGit() (string, error) {
	b, err := exec.Command("git", "describe", "--always", "--tags", "--dirty").Output()
	if err != nil {
		return "", err
	}
	return strings.Trim(string(b), " \n\t"), nil
}

func doUIBuild() error {
	errors := make(chan error, 2)
	wg := sync.WaitGroup{}
	for _, task := range []string{"./compile-js", "./compile-css"} {
		wg.Add(1)
		go func(t string) {
			log.Println("Running UI Build Task: ", t)
			cmd := exec.Command(t)
			uiEnv(cmd)
			errors <- cmd.Run()
			wg.Done()
		}(task)
	}
	go func() {
		wg.Wait()
		close(errors)
	}()
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func uiEnv(cmd *exec.Cmd) {
	cmd.Dir = filepath.Join(pwd(), "ui")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
}

func pwd() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}
