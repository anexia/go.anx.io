package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"

	git "github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	gitPlumbing "github.com/go-git/go-git/v5/plumbing"

	"github.com/Masterminds/semver/v3"
)

type VersionedFileReader interface {
	ReadFile(path, version string) ([]byte, error)
}

type repositoryReader struct {
	repository *git.Repository
	versions   map[string]*gitPlumbing.Reference
}

func (r repositoryReader) ReadFile(path, version string) ([]byte, error) {
	tag, ok := r.versions[version]
	if !ok {
		return nil, fmt.Errorf("no tag for version in repository: %w", git.ErrTagNotFound)
	}

	commit, err := r.repository.CommitObject(tag.Hash())
	if err != nil {
		return nil, fmt.Errorf("error resolving tag '%v' to commit: %w", tag.Hash(), err)
	}

	tree, err := r.repository.TreeObject(commit.TreeHash)
	if err != nil {
		return nil, fmt.Errorf("error retrieving tree for revision '%v': %w", tag.Hash(), err)
	}

	entry, err := tree.FindEntry(path)
	if err != nil {
		return nil, fmt.Errorf("cannot find path in given versions tree: %w", err)
	}

	file, err := tree.TreeEntryFile(entry)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve file from given versions tree: %w", err)
	}

	contents, err := file.Contents()
	return []byte(contents), err
}

func loadSources(cachePath string) error {
	stat, err := os.Stat(cachePath)
	if errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(cachePath, os.ModeDir|0755)

		if err == nil {
			stat, err = os.Stat(cachePath)
		}
	}

	if err != nil {
		return fmt.Errorf("error stat'ing cache path: %w", err)
	} else {
		if !stat.IsDir() {
			return fmt.Errorf("%w: cache path is not a directory", os.ErrInvalid)
		}
	}

	for _, pkg := range pkgConfig {
		if err := loadSource(cachePath, pkg); err != nil {
			return err
		}
	}

	return nil
}

func loadSource(cachePath string, pkg *pkgEntry) error {
	source, err := url.Parse(pkg.Source)
	if err != nil {
		return fmt.Errorf("invalid source url: %w", err)
	}

	localPath := path.Join(cachePath, source.Host, source.Path)
	repo, err := git.PlainOpen(localPath)

	if err == git.ErrRepositoryNotExists {
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

	repo.Fetch(&git.FetchOptions{
		Force:    true,
		RefSpecs: []gitConfig.RefSpec{"refs/heads/*:refs/heads/*"},
	})

	if iter, err := repo.Tags(); err != nil {
		return fmt.Errorf("error iterating tags in local git repository: %w", err)
	} else {
		type tagVersion struct {
			tag     *gitPlumbing.Reference
			version *semver.Version
		}

		tags := make([]tagVersion, 0)
		err := iter.ForEach(func(tag *gitPlumbing.Reference) error {
			// first we check if we have a tree for this tag
			if commit, err := repo.CommitObject(tag.Hash()); err != nil {
				return nil
			} else {
				if _, err := repo.TreeObject(commit.TreeHash); err != nil {
					return nil
				}
			}

			// parse tag as semver to filter on only release tags
			if v, err := semver.NewVersion(tag.Name().Short()); err == nil {
				tags = append(tags, tagVersion{
					tag:     tag,
					version: v,
				})
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
			sort.Slice(tags, func(i, j int) bool {
				// we sort highest-to-lowest
				return tags[i].version.GreaterThan(tags[j].version)
			})

			for _, v := range tags {
				pkg.Versions = append(pkg.Versions, v.version.String())

				fileReader.versions[v.version.String()] = v.tag
			}
		}

		branchIterator, err := repo.Branches()
		if err != nil {
			return fmt.Errorf("error iterating branches in local git repository: %w", err)
		}

		err = branchIterator.ForEach(func(branch *gitPlumbing.Reference) error {
			pkg.Versions = append(pkg.Versions, branch.Name().Short())
			fileReader.versions[branch.Name().Short()] = branch
			return nil
		})
		if err != nil {
			return fmt.Errorf("error iterating branches in local git repository: %w", err)
		}

		pkg.FileReader = &fileReader

		if pkg.Summary == "" {
			contents, err := pkg.FileReader.ReadFile("README.md", pkg.Versions[0])
			if err == nil {
				pkg.Summary, _ = extractFirstHeader(string(contents))
				// ignoring error on purpose - it either works or we don't have a summary
			}
		}

		return nil
	}
}
