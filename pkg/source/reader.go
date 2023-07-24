package source

import (
	"errors"
	"fmt"
	"path"
	"regexp"

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
