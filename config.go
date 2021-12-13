package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type pkgEntry struct {
	Source     string `yaml:"source"`
	TargetName string `yaml:"targetName"`
	Summary    string `yaml:"summary"`

	Versions   []string            `json:"-"`
	FileReader VersionedFileReader `json:"-"`
}

var pkgConfig []*pkgEntry

func loadPackageConfig(filePath string) error {
	if file, err := os.Open(filePath); err != nil {
		return err
	} else {
		if err := yaml.NewDecoder(file).Decode(&pkgConfig); err != nil {
			return err
		}
	}

	for _, pkg := range pkgConfig {
		pkg.Versions = make([]string, 0)

		if pkg.TargetName == "" {
			source, err := url.Parse(pkg.Source)
			if err != nil {
				return fmt.Errorf("error parsing source url '%v': %w", pkg.Source, err)
			}

			pkg.TargetName = strings.TrimSuffix(path.Base(source.Path), ".git")
		}
	}

	return nil
}
