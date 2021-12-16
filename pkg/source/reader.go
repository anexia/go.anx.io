package source

import (
	"fmt"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	gitPlumbing "github.com/go-git/go-git/v5/plumbing"

	"github.com/Masterminds/semver/v3"
)

// repositoryReader is an implementation of VersionedFileReader for git repositories.
type repositoryReader struct {
	repository *git.Repository
	versions   map[string]*gitPlumbing.Reference
}

// ReadFile implements VersionedFileReader
func (r repositoryReader) ReadFile(path, version string) ([]byte, error) {
	tag, ok := r.versions[version]
	if !ok {
		return nil, fmt.Errorf("no tag for version in repository: %w", git.ErrTagNotFound)
	}

	var commitHash gitPlumbing.Hash
	if tagObject, err := r.repository.TagObject(tag.Hash()); err != nil {
		commitHash = tag.Hash()
	} else {
		commitHash = tagObject.Target
	}

	commit, err := r.repository.CommitObject(commitHash)
	if err != nil {
		return nil, fmt.Errorf("error resolving commit hash '%v' to commit: %w", commitHash, err)
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

func (r repositoryReader) Versions() []string {
	ret := make([]string, 0, len(r.versions))

	for v := range r.versions {
		ret = append(ret, v)
	}

	sort.Slice(ret, func(a int, b int) bool {
		verA, errA := semver.NewVersion(ret[a])
		verB, errB := semver.NewVersion(ret[b])

		if errA == nil && errB != nil {
			return true
		} else if errB == nil && errA != nil {
			return false
		} else if verA == verB && verA == nil {
			if ret[a] == "main" {
				return true
			} else if ret[b] == "main" {
				return false
			} else if ret[a] == "master" {
				return false
			}

			return strings.Compare(ret[a], ret[b]) > 0
		} else {
			return verA.Compare(verB) > 0
		}
	})

	return ret
}
