// This is a small helper tool using go's vcs helper to resolve our vanity import path
// against a directory of generated html files.

package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/tools/go/vcs"
	"gopkg.in/yaml.v3"
)

type FileRoundTripper struct {
	dir string
}

func (frt *FileRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	notFoundResponse := bufio.NewReader(strings.NewReader("HTTP/1.0 404 Not Found\r\n\r\n"))
	serverErrorResponse := bufio.NewReader(strings.NewReader("HTTP/1.0 500 Internal Server Error\r\n\r\n"))
	successResponse := bufio.NewReader(strings.NewReader("HTTP/1.0 200 OK\r\n\r\n"))

	sourceFile := req.URL.Path

	if stat, err := os.Stat(path.Join(frt.dir, sourceFile)); err != nil {
		return http.ReadResponse(notFoundResponse, req)
	} else if stat.IsDir() {
		sourceFile = path.Join(sourceFile, "index.html")
		if stat, err := os.Stat(path.Join(frt.dir, sourceFile)); err != nil {
			return http.ReadResponse(notFoundResponse, req)
		} else if stat.IsDir() {
			return http.ReadResponse(notFoundResponse, req)
		}
	}

	var responseData io.Reader

	if r, err := os.Open(path.Join(frt.dir, sourceFile)); err != nil {
		return http.ReadResponse(serverErrorResponse, req)
	} else {
		responseData = io.MultiReader(successResponse, r)
	}

	return http.ReadResponse(bufio.NewReader(responseData), req)
}

func main() {
	type pkgDef struct {
		Source     string `yaml:"source"`
		TargetName string `yaml:"targetName"`
	}

	htmlPath := os.Args[1]
	configPath := os.Args[2]

	pkgs := []*pkgDef{}

	if r, err := os.Open(configPath); err != nil {
		fmt.Printf("Error opening config file %q: %v", configPath, err)
		os.Exit(-1)
	} else {
		if err := yaml.NewDecoder(r).Decode(&pkgs); err != nil {
			fmt.Printf("Error parsing config file %q: %v", configPath, err)
			os.Exit(-1)
		}
	}

	fileTransport := FileRoundTripper{dir: htmlPath}

	transport := http.Transport{}

	// handle http and https with our file transport
	transport.RegisterProtocol("http", &fileTransport)
	transport.RegisterProtocol("https", &fileTransport)

	// disable automatically trying to upgrade to HTTP/2 - which errors since we registered https already
	transport.TLSNextProto = make(map[string]func(auth string, c *tls.Conn) http.RoundTripper)

	// configure default client to use this transport
	http.DefaultClient.Transport = &transport

	for _, pkg := range pkgs {
		if pkg.TargetName == "" {
			if u, err := url.Parse(pkg.Source); err != nil {
				fmt.Printf("Error parsing source url %q: %v", pkg.Source, err)
				os.Exit(-1)
			} else {
				pkg.TargetName = path.Base(u.Path)
				pkg.TargetName = strings.TrimSuffix(pkg.TargetName, path.Ext(pkg.TargetName))
			}
		}

		checkPath := path.Join("go.anx.io", pkg.TargetName)

		repo, err := vcs.RepoRootForImportDynamic(checkPath, false)
		if err != nil {
			fmt.Printf("Error retrieving repo root for import path %q: %v", checkPath, err)
			os.Exit(-1)
		}

		if repo.VCS.Name != "Git" {
			fmt.Printf("Unexpected VCS for %q: %q", checkPath, repo.VCS.Name)
			os.Exit(-1)
		}

		if repo.Repo != pkg.Source {
			fmt.Printf("Unexpected repo for %q: %q", checkPath, repo.Repo)
			os.Exit(-1)
		}
	}
}
