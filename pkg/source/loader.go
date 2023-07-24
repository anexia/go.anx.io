package source

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	git "github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	gitPlumbing "github.com/go-git/go-git/v5/plumbing"

	"github.com/Masterminds/semver/v3"

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

	err = repo.Fetch(&git.FetchOptions{
		Force:    true,
		RefSpecs: []gitConfig.RefSpec{"refs/heads/*:refs/heads/*"},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("error fetching source repository: %w", err)
	}

	iter, err := repo.Tags()
	if err != nil {
		return fmt.Errorf("error iterating tags in local git repository: %w", err)
	}

	type tagVersion struct {
		tag     *gitPlumbing.Reference
		version string
	}

	tags := make([]tagVersion, 0)

	err = iter.ForEach(func(tag *gitPlumbing.Reference) error {
		var commit gitPlumbing.Hash
		if tagObject, err := repo.TagObject(tag.Hash()); err != nil {
			commit = tag.Hash()
		} else {
			commit = tagObject.Target
		}

		// first we check if we have a tree for this tag
		if commit, err := repo.CommitObject(commit); err != nil {
			log.Printf("Not using tag %v since we do not have a commit for it (%v)", tag.Name().Short(), err)
			return nil
		} else if _, err := repo.TreeObject(commit.TreeHash); err != nil {
			log.Printf("Not using tag %v since we do not have a tree for its commit", tag.Name().Short())
			return nil //nolint:nilerr
		}

		// parse tag as semver to filter on only release tags
		if _, err := semver.NewVersion(strings.TrimPrefix(tag.Name().Short(), "v")); err == nil {
			tags = append(tags, tagVersion{
				tag:     tag,
				version: tag.Name().Short(),
			})
		} else {
			log.Printf("Not using tag %v due to error %v", tag.Name().Short(), err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating tags in local git repository: %w", err)
	}

	fileReader := repositoryReader{
		repository: repo,
		versions:   make(map[string]*gitPlumbing.Reference),
	}

	if len(tags) > 0 {
		for _, v := range tags {
			fileReader.versions[v.version] = v.tag
		}
	}

	branchIterator, err := repo.Branches()
	if err != nil {
		return fmt.Errorf("error iterating branches in local git repository: %w", err)
	}

	err = branchIterator.ForEach(func(branch *gitPlumbing.Reference) error {
		fileReader.versions[branch.Name().Short()] = branch
		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating branches in local git repository: %w", err)
	}

	if err := fileReader.readMajorVersions(); err != nil {
		return fmt.Errorf("failed to retrieve versions of package '%v': %w", pkg.TargetName, err)
	}

	pkg.FileReader = &fileReader

	if pkg.Summary == "" {
		contents, err := fileReader.ReadFile("README.md", fileReader.Versions(fileReader.MajorVersions()[0])[0])
		if err == nil {
			pkg.Summary = markdown.ExtractFirstHeader(contents)
		}
	}

	log.Printf("Loaded package '%v' with %v versions", pkg.TargetName, len(fileReader.versions))

	return nil
}
