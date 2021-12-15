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

	if err := loadTemplates(*templateDirPath); err != nil {
		log.Fatalf("Error loading templates from directory '%v': %v", *templateDirPath, err)
	}

	if err := loadPackageConfig(*configFile); err != nil {
		log.Fatalf("Error loading config file '%v': %v", *configFile, err)
	}

	if err := loadSources(*sourceCache); err != nil {
		log.Fatalf("Error loading sources: %v", err)
	}

	for _, pkg := range pkgConfig {
		log.Printf("Loaded package '%v' with %v versions", pkg.TargetName, len(pkg.Versions))
	}

	switch *mode {
	case "serve":
		listenAndServe(*staticDirPath, *listenAddress)
	case "generate":
		if err := renderFiles(*destinationPath); err != nil {
			log.Fatalf("Error rendering files: %v", err)
		}

		if err := copyStaticFiles(*staticDirPath, *destinationPath); err != nil {
			log.Fatalf("Error copying static files: %v", err)
		}
	}
}

func handlePackage(pkg *pkgEntry) {
	basePath := "/"

	if pkg != nil {
		basePath = fmt.Sprintf("/%s/", pkg.TargetName)
	}

	http.HandleFunc(basePath, func(res http.ResponseWriter, req *http.Request) {
		filePath := strings.TrimPrefix(req.URL.Path, basePath)

		buffer := bytes.Buffer{}
		if err := renderFile(pkg, filePath, &buffer); err != nil {
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

func listenAndServe(staticFilePath, listenAddress string) {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticFilePath))))

	for _, pkg := range pkgConfig {
		handlePackage(pkg)
	}

	handlePackage(nil)

	err := http.ListenAndServe(listenAddress, nil)
	if err != nil {
		log.Fatalf("Error starting http server: %v", err)
	}
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
