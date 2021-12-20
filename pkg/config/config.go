package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/anexia-it/go.anx.io/pkg/types"
)

func Load(filePath string) ([]*types.Package, error) {
	ret := []*types.Package{}

	if file, err := os.Open(filePath); err != nil {
		return nil, fmt.Errorf("error opening config file %q: %w", filePath, err)
	} else if err := yaml.NewDecoder(file).Decode(&ret); err != nil {
		return nil, fmt.Errorf("error decoding config file %q: %w", filePath, err)
	}

	for _, pkg := range ret {
		if pkg.TargetName == "" {
			source, err := url.Parse(pkg.Source)
			if err != nil {
				return nil, fmt.Errorf("error parsing source url '%v': %w", pkg.Source, err)
			}

			pkg.TargetName = strings.TrimSuffix(path.Base(source.Path), ".git")
		}
	}

	return ret, nil
}
