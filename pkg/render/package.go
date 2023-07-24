package render

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
	"github.com/anexia-it/go.anx.io/pkg/types"
)

type packageTemplateData struct {
	layoutTemplateData

	Package        *types.Package
	CurrentVersion string
	MajorVersion   string
}

func (r *Renderer) renderPackageFile(pkg *types.Package, filePath string, writer io.Writer) error {
	var (
		moduleVersions []string
		majorVersion   string
	)

	majorVersionPrefixRegex := regexp.MustCompile(`^v\d+(/|$)`)
	if mv := majorVersionPrefixRegex.FindString(filePath); mv != "" {
		majorVersion = strings.TrimSuffix(mv, "/")
		moduleVersions = pkg.FileReader.Versions(majorVersion)

		if len(moduleVersions) > 0 {
			filePath = strings.TrimPrefix(filePath, mv)
		}
	}

	if len(moduleVersions) == 0 {
		moduleVersions = pkg.FileReader.Versions("")
	}

	pathAndVersion := strings.SplitN(filePath, "@", 2)
	filePath = pathAndVersion[0]

	version := moduleVersions[0]

	if len(pathAndVersion) == 2 {
		version = pathAndVersion[1]
	}

	if filePath == "" || filePath == "index.html" {
		filePath = "README.md"
	}

	content, err := pkg.FileReader.ReadFile(filePath, version)
	if err != nil {
		return fmt.Errorf("error reading file '%v' of version '%v': %w", filePath, version, err)
	}

	content, err = markdownContent(content, filePath)
	if err != nil {
		return fmt.Errorf("error retrieving markdown for package file: %w", err)
	}

	data := packageTemplateData{
		layoutTemplateData: layoutTemplateData{
			Title:           markdown.ExtractFirstHeader(content),
			MarkdownContent: content,
			CurrentFile:     filePath,
		},
		Package:        pkg,
		CurrentVersion: version,
		MajorVersion:   majorVersion,
	}

	return r.executeTemplate(writer, "package.tmpl", data)
}

func (r *Renderer) filesForPackage(pkg *types.Package) []string {
	// files we want for every version
	versionedFiles := []string{"README.md"}

	majorVersions := pkg.FileReader.MajorVersions()

	ret := make([]string, 0)

	// for every major version we generate version "" (latest version) and all specific versions of it
	for _, major := range majorVersions {
		versions := []string{""}
		versions = append(versions, pkg.FileReader.Versions(major)...)

		majorFiles := make([]string, 0, (len(versions)*len(versionedFiles))+1)

		// we always want index without version suffix
		majorFiles = append(majorFiles, path.Join(major, "index.html"))

		for _, v := range versions {
			for _, filename := range versionedFiles {
				if v != "" {
					filename = fmt.Sprintf("%v@%v", filename, v)
				}

				majorFiles = append(majorFiles, path.Join(major, filename))
			}
		}

		ret = append(ret, majorFiles...)
	}

	return ret
}
