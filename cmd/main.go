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
	"time"

	"github.com/anexia-it/go.anx.io/pkg/config"
	"github.com/anexia-it/go.anx.io/pkg/render"
	"github.com/anexia-it/go.anx.io/pkg/source"
	"github.com/anexia-it/go.anx.io/pkg/types"
)

var version = "dev"
var sourceURL = ""

var (
	mode            = "generate"
	configFile      = "packages.yaml"
	templateDirPath = ""
	contentPath     = ""
	staticDirPath   = ""
	sourceCache     = "source-cache"
	listenAddress   = "localhost:1312"
	destinationPath = "public"
)

func main() {
	if workdir, err := os.Getwd(); err == nil {
		templateDirPath = path.Join(workdir, "templates")
		staticDirPath = path.Join(workdir, "static")
		contentPath = path.Join(workdir, "content")
	} else {
		log.Printf("Cannot determine current working directory: %v", err)
	}

	flag.StringVar(&mode, "mode", mode, "Mode to run this into (generate|serve)")
	flag.StringVar(&configFile, "config-file", configFile, "Path to config file to use")
	flag.StringVar(&templateDirPath, "template-directory", templateDirPath, "Path to directory containing the templates")
	flag.StringVar(&contentPath, "content-directory", contentPath, "Path to directory containing the content files")
	flag.StringVar(&staticDirPath, "static-directory", staticDirPath, "Path to directory containing the static files")
	flag.StringVar(&sourceCache, "source-cache", sourceCache, "Path to where to cache sources")
	flag.StringVar(&listenAddress, "listen-address", listenAddress, "Address to listen on in serve mode")
	flag.StringVar(&destinationPath, destinationPath, "public", "Path to directory to store the generated files in")

	flag.Parse()

	if mode != "generate" && mode != "serve" {
		flag.Usage()
		return
	}

	packages, err := config.Load(configFile)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	sourceLoader, err := source.NewLoader(sourceCache)
	if err != nil {
		log.Fatalf("Error initializing source loader: %v", err)
	}

	if err := sourceLoader.LoadSources(packages); err != nil {
		log.Fatalf("Error loading sources: %v", err)
	}

	renderer, err := render.NewRenderer(templateDirPath, contentPath, packages)
	if err != nil {
		log.Fatalf("Error initializing Renderer: %v", err)
	}

	renderer.SetBuildInfo(version, sourceURL)

	switch mode {
	case "serve":
		runServe(packages, renderer)
	case "generate":
		runGenerate(renderer)
	}
}

func runServe(packages []*types.Package, renderer *render.Renderer) {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDirPath))))

	for _, pkg := range packages {
		servePackage(pkg, renderer)
	}

	servePackage(nil, renderer)

	//nolint:exhaustruct // We only set useful things here
	server := http.Server{
		Addr:              listenAddress,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Error starting http server: %v", err)
	}
}

func runGenerate(renderer *render.Renderer) {
	if err := renderer.GenerateFiles(destinationPath); err != nil {
		log.Fatalf("Error rendering files: %v", err)
	}

	if err := copyStaticFiles(staticDirPath, destinationPath); err != nil {
		log.Fatalf("Error copying static files: %v", err)
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

	err := fs.WalkDir(staticFiles, ".", func(walkEntry string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatalf("Error walking static files directory: %v", err)
		}

		if d.IsDir() {
			err := os.MkdirAll(path.Join(destinationPath, "static", walkEntry), 0755)
			if err != nil {
				return fmt.Errorf("error creating directory %q: %w", destinationPath, err)
			}

			return nil
		}

		sourceStream, err := os.Open(path.Join(staticPath, walkEntry))
		if err != nil {
			return fmt.Errorf("error opening source file %q: %w", staticPath, err)
		}

		//nolint:nosnakecase // O_WRONLY, O_CREATE and O_TRUNC are defined by the os package (and underlying POSIX spec).
		destinationStream, err := os.OpenFile(path.Join(destinationPath, "static", walkEntry), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("error opening destination file %q: %w", destinationPath, err)
		}

		_, err = io.Copy(destinationStream, sourceStream)
		if err != nil {
			return fmt.Errorf("error copying file data: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking static files directory: %w", err)
	}

	return nil
}
