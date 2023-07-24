package source

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"

	git "github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
	"github.com/anexia-it/go.anx.io/pkg/types"
)

type Loader struct {
	cachePath string
}

func NewLoader(cachePath string) (*Loader, error) {
	stat, err := os.Stat(cachePath)
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(cachePath, os.ModeDir|0755)

		if err == nil {
			stat, err = os.Stat(cachePath)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("error stat'ing cache path: %w", err)
	} else if !stat.IsDir() {
		return nil, fmt.Errorf("%w: cache path is not a directory", os.ErrInvalid)
	}

	return &Loader{
		cachePath: cachePath,
	}, nil
}

func (l *Loader) LoadSources(pkgs []*types.Package) error {
	for _, pkg := range pkgs {
		if err := l.loadSource(pkg); err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) loadSource(pkg *types.Package) error {
	source, err := url.Parse(pkg.Source)
	if err != nil {
		return fmt.Errorf("invalid source url: %w", err)
	}

	localPath := path.Join(l.cachePath, source.Host, source.Path)
	repo, err := git.PlainOpen(localPath)

	if errors.Is(err, git.ErrRepositoryNotExists) {
		//nolint:exhaustruct
		repo, err = git.PlainClone(localPath, false, &git.CloneOptions{
			URL:        source.String(),
			NoCheckout: true,
		})
		if err != nil {
			return fmt.Errorf("error cloning source repository: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error opening local git repository: %w", err)
	}

	//nolint:exhaustruct
	err = repo.Fetch(&git.FetchOptions{
		Force:    true,
		RefSpecs: []gitConfig.RefSpec{"refs/heads/*:refs/heads/*"},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("error fetching source repository: %w", err)
	}

	reader, err := newRepositoryReader(repo)
	if err != nil {
		return fmt.Errorf("error reading repository metadata for package %q: %w", pkg.TargetName, err)
	}

	pkg.FileReader = reader

	if pkg.Summary == "" {
		contents, err := reader.ReadFile("README.md", reader.Versions(reader.MajorVersions()[0])[0])
		if err == nil {
			pkg.Summary = markdown.ExtractFirstHeader(contents)
		}
	}

	log.Printf("Loaded package '%v' with %v versions", pkg.TargetName, len(reader.versions))

	return nil
}
