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

func main() {
	workdir, err := os.Getwd()
	if err != nil {
		log.Printf("Cannot determine current working directory: %v", err)
		workdir = ""
	}

	templateDirDefault := ""
	staticDirDefault := ""
	if workdir != "" {
		templateDirDefault = path.Join(workdir, "templates")
		staticDirDefault = path.Join(workdir, "static")
	}

	mode := flag.String("mode", "generate", "Mode to run this into (generate|serve)")
	configFile := flag.String("config-file", "packages.yaml", "Path to config file to use")
	templateDirPath := flag.String("template-directory", templateDirDefault, "Path to directory containing the templates")
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

	renderer, err := render.NewRenderer(*templateDirPath, packages)
	if err != nil {
		log.Fatalf("Error initializing Renderer: %v", err)
	}

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
		if err := renderFiles(renderer, packages, *destinationPath); err != nil {
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
			res.Write([]byte(err.Error()))
		} else {
			if strings.HasSuffix(filePath, ".css") {
				res.Header().Add("Content-Type", "text/css; charset=utf-8")
			} else {
				res.Header().Add("Content-Type", "text/html; charset=utf-8")
			}

			res.WriteHeader(http.StatusOK)
			res.Write(buffer.Bytes())
		}
	})
}

func renderFiles(renderer *render.Renderer, pkgs []*types.Package, destinationPath string) error {
	for _, pkg := range pkgs {
		pkgPath := path.Join(destinationPath, pkg.TargetName)
		if err := os.MkdirAll(pkgPath, os.ModeDir|0755); err != nil {
			return err
		}

		// we generate "unversioned/latest" and all defined versions
		versions := []string{""}
		versions = append(versions, pkg.FileReader.Versions()...)

		// lists the files we want for every version
		versionedFiles := []string{"README.md"}

		files := make([]string, 0, (len(versions)*len(versionedFiles))+1)
		files = append(files, "") // we always want index

		for _, v := range versions {
			for _, vf := range versionedFiles {
				if v != "" {
					files = append(files, fmt.Sprintf("%v@%v", vf, v))
				} else {
					files = append(files, vf)
				}
			}
		}

		for _, f := range files {
			// Every "file" is actually a directory in which we place an index.html.
			// This is needed since we want `README.md` in path, but serving HTML. On GitHub Pages we cannot
			// configure the MIME type, so we have to serve a file with html extension - our `index.html`.
			destinationDirectory := path.Join(pkgPath, f)
			err := os.MkdirAll(destinationDirectory, 0755)
			if err != nil {
				return err
			}

			destinationFile := path.Join(destinationDirectory, "index.html")
			w, err := os.OpenFile(destinationFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}

			err = renderer.RenderFile(pkg, f, w)
			if err != nil {
				return err
			}
		}
	}

	if contentFiles, err := fs.Glob(os.DirFS("content"), "*"); err != nil {
		return nil
	} else {
		for _, f := range contentFiles {
			if f == "index.md" {
				f = ""
			}

			// Every "file" is actually a directory in which we place an index.html.
			// This is needed since we want `README.md` in path, but serving HTML. On GitHub Pages we cannot
			// configure the MIME type, so we have to serve a file with html extension - our `index.html`.
			destinationDirectory := path.Join(destinationPath, f)
			err := os.MkdirAll(destinationDirectory, 0755)
			if err != nil {
				return err
			}

			destinationFile := path.Join(destinationDirectory, "index.html")
			w, err := os.OpenFile(destinationFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}

			err = renderer.RenderFile(nil, f, w)
			if err != nil {
				return err
			}
		}
	}

	generatedStaticFiles := []string{"chroma/style.css"}
	for _, f := range generatedStaticFiles {
		destinationDirectory := path.Join(destinationPath, path.Dir(f))
		err := os.MkdirAll(destinationDirectory, 0755)
		if err != nil {
			return err
		}

		destinationFile := path.Join(destinationPath, f)
		w, err := os.OpenFile(destinationFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		err = renderer.RenderFile(nil, f, w)
		if err != nil {
			return err
		}
	}

	return nil
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
