package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/anexia-it/go.anx.io/pkg/config"
	"github.com/anexia-it/go.anx.io/pkg/render"
	"github.com/anexia-it/go.anx.io/pkg/source"
	"github.com/anexia-it/go.anx.io/pkg/types"
)

var version = "dev"
var sourceURL = ""

func main() {
	workdir, err := os.Getwd()
	if err != nil {
		log.Printf("Cannot determine current working directory: %v", err)
		workdir = ""
	}

	templateDirDefault := ""
	staticDirDefault := ""
	contentPathDefault := ""
	if workdir != "" {
		templateDirDefault = path.Join(workdir, "templates")
		staticDirDefault = path.Join(workdir, "static")
		contentPathDefault = path.Join(workdir, "content")
	}

	mode := flag.String("mode", "generate", "Mode to run this into (generate|serve)")
	configFile := flag.String("config-file", "packages.yaml", "Path to config file to use")
	templateDirPath := flag.String("template-directory", templateDirDefault, "Path to directory containing the templates")
	contentPath := flag.String("content-directory", contentPathDefault, "Path to directory containing the content files")
	staticDirPath := flag.String("static-directory", staticDirDefault, "Path to directory containing the static files")
	sourceCache := flag.String("source-cache", "source-cache", "Path to where to cache sources")
	listenAddress := flag.String("listen-address", "localhost:1312", "Address to listen on in serve mode")
	destinationPath := flag.String("destination-path", "public", "Path to directory to store the generated files in")

	flag.Parse()

	if *mode != "generate" && *mode != "serve" {
		flag.Usage()
		return
	}

	packages, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	sourceLoader, err := source.NewLoader(*sourceCache)
	if err != nil {
		log.Fatalf("Error initializing source loader: %v", err)
	}

	if err := sourceLoader.LoadSources(packages); err != nil {
		log.Fatalf("Error loading sources: %v", err)
	}

	renderer, err := render.NewRenderer(*templateDirPath, *contentPath, packages)
	if err != nil {
		log.Fatalf("Error initializing Renderer: %v", err)
	}
	renderer.SetBuildInfo(version, sourceURL)

	switch *mode {
	case "serve":
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(*staticDirPath))))

		for _, pkg := range packages {
			servePackage(pkg, renderer)
		}

		servePackage(nil, renderer)

		err := http.ListenAndServe(*listenAddress, nil)
		if err != nil {
			log.Fatalf("Error starting http server: %v", err)
		}
	case "generate":
		if err := renderer.GenerateFiles(*destinationPath); err != nil {
			log.Fatalf("Error rendering files: %v", err)
		}

		if err := copyStaticFiles(*staticDirPath, *destinationPath); err != nil {
			log.Fatalf("Error copying static files: %v", err)
		}
	}
}

func servePackage(pkg *types.Package, renderer *render.Renderer) {
	basePath := "/"

	if pkg != nil {
		basePath = fmt.Sprintf("/%s/", pkg.TargetName)
	}

	http.HandleFunc(basePath, func(res http.ResponseWriter, req *http.Request) {
		filePath := strings.TrimPrefix(req.URL.Path, basePath)

		buffer := bytes.Buffer{}
		if err := renderer.RenderFile(pkg, filePath, &buffer); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			_, _ = res.Write([]byte(err.Error()))
		} else {
			if strings.HasSuffix(filePath, ".css") {
				res.Header().Add("Content-Type", "text/css; charset=utf-8")
			} else {
				res.Header().Add("Content-Type", "text/html; charset=utf-8")
			}

			res.WriteHeader(http.StatusOK)
			_, _ = res.Write(buffer.Bytes())
		}
	})
}

func copyStaticFiles(staticPath string, destinationPath string) error {
	staticFiles := os.DirFS(staticPath)
	return fs.WalkDir(staticFiles, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatalf("Error walking static files directory: %v", err)
		}

		if d.IsDir() {
			err := os.MkdirAll(path.Join(destinationPath, "static", p), 0755)
			if err != nil {
				return err
			}
		} else {
			r, err := os.Open(path.Join(staticPath, p))
			if err != nil {
				return err
			}

			w, err := os.OpenFile(path.Join(destinationPath, "static", p), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}

			_, err = io.Copy(w, r)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
