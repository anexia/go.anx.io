package source

import (
	"errors"
	"fmt"
	"log"
	"path"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	git "github.com/go-git/go-git/v5"
	gitPlumbing "github.com/go-git/go-git/v5/plumbing"
	gitObject "github.com/go-git/go-git/v5/plumbing/object"

	"golang.org/x/mod/modfile"
)

// repositoryReader is an implementation of VersionedFileReader for git repositories.
type repositoryReader struct {
	repository    *git.Repository
	majorVersions map[string][]string
	versions      map[string]*gitPlumbing.Reference
}

func newRepositoryReader(repo *git.Repository) (*repositoryReader, error) {
	ret := repositoryReader{
		repository:    repo,
		versions:      make(map[string]*gitPlumbing.Reference),
		majorVersions: make(map[string][]string),
	}

	if err := ret.addTagVersions(); err != nil {
		return nil, fmt.Errorf("failed adding versions based on tags: %w", err)
	}

	if err := ret.addBranchVersions(); err != nil {
		return nil, fmt.Errorf("failed adding versions based on branches: %w", err)
	}

	if err := ret.readMajorVersions(); err != nil {
		return nil, fmt.Errorf("failed to retrieve versions: %w", err)
	}

	return &ret, nil
}

func (r *repositoryReader) addTagVersions() error {
	iter, err := r.repository.Tags()
	if err != nil {
		return fmt.Errorf("error iterating tags in local git repository: %w", err)
	}

	err = iter.ForEach(func(tag *gitPlumbing.Reference) error {
		var commit gitPlumbing.Hash
		if tagObject, err := r.repository.TagObject(tag.Hash()); err != nil {
			commit = tag.Hash()
		} else {
			commit = tagObject.Target
		}

		// first we check if we have a tree for this tag
		if commit, err := r.repository.CommitObject(commit); err != nil {
			log.Printf("Not using tag %v since we do not have a commit for it (%v)", tag.Name().Short(), err)
			return nil
		} else if _, err := r.repository.TreeObject(commit.TreeHash); err != nil {
			log.Printf("Not using tag %v since we do not have a tree for its commit (%v)", tag.Name().Short(), err)
			return nil
		}

		// parse tag as semver to filter on only release tags
		if _, err := semver.NewVersion(strings.TrimPrefix(tag.Name().Short(), "v")); err == nil {
			r.versions[tag.Name().Short()] = tag
		} else {
			log.Printf("Not using tag %v due to error %v", tag.Name().Short(), err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating tags in local git repository: %w", err)
	}

	return nil
}

func (r *repositoryReader) addBranchVersions() error {
	branchIterator, err := r.repository.Branches()
	if err != nil {
		return fmt.Errorf("error iterating branches in local git repository: %w", err)
	}

	err = branchIterator.ForEach(func(branch *gitPlumbing.Reference) error {
		r.versions[branch.Name().Short()] = branch
		return nil
	})
	if err != nil {
		return fmt.Errorf("error iterating branches in local git repository: %w", err)
	}

	return nil
}

func (r *repositoryReader) readMajorVersions() error {
	r.majorVersions = make(map[string][]string)

	for version := range r.versions {
		fileContents, err := r.ReadFile("go.mod", version)
		if isNotFound := errors.Is(err, gitObject.ErrEntryNotFound); err != nil && !isNotFound {
			return err
		} else if isNotFound {
			continue
		}

		file, err := modfile.Parse(fmt.Sprintf("go.mod@%v", version), []byte(fileContents), nil)
		if err != nil {
			return fmt.Errorf("error parsing go.mod file: %w", err)
		}

		maybeMajor := path.Base(file.Module.Mod.Path)

		if !regexp.MustCompile(`^v\d+$`).MatchString(maybeMajor) {
			maybeMajor = ""
		}

		if _, ok := r.majorVersions[maybeMajor]; !ok {
			r.majorVersions[maybeMajor] = make([]string, 0)
		}

		r.majorVersions[maybeMajor] = append(r.majorVersions[maybeMajor], version)
	}

	for major := range r.majorVersions {
		sortVersions(r.majorVersions[major])
	}

	return nil
}

// ReadFile implements VersionedFileReader on repositoryReader.
func (r repositoryReader) ReadFile(path, version string) (string, error) {
	tag, ok := r.versions[version]
	if !ok {
		return "", fmt.Errorf("no tag for version in repository: %w", git.ErrTagNotFound)
	}

	var commitHash gitPlumbing.Hash
	if tagObject, err := r.repository.TagObject(tag.Hash()); err != nil {
		commitHash = tag.Hash()
	} else {
		commitHash = tagObject.Target
	}

	commit, err := r.repository.CommitObject(commitHash)
	if err != nil {
		return "", fmt.Errorf("error resolving commit hash '%v' to commit: %w", commitHash, err)
	}

	tree, err := r.repository.TreeObject(commit.TreeHash)
	if err != nil {
		return "", fmt.Errorf("error retrieving tree for revision '%v': %w", tag.Hash(), err)
	}

	entry, err := tree.FindEntry(path)
	if err != nil {
		return "", fmt.Errorf("cannot find path in given versions tree: %w", err)
	}

	file, err := tree.TreeEntryFile(entry)
	if err != nil {
		return "", fmt.Errorf("cannot retrieve file from given versions tree: %w", err)
	}

	contents, err := file.Contents()
	if err != nil {
		return "", fmt.Errorf("error reading file contents: %w", err)
	}

	return contents, nil
}

func (r repositoryReader) MajorVersions() []string {
	ret := make([]string, 0, len(r.majorVersions))

	for v := range r.majorVersions {
		ret = append(ret, v)
	}

	sortVersions(ret)

	return ret
}

func (r repositoryReader) Versions(major string) []string {
	if v, ok := r.majorVersions[major]; ok {
		return v
	}

	return nil
}
