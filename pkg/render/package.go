package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/anexia-it/go.anx.io/pkg/types"
)

type packageTemplateData struct {
	layoutTemplateData

	Package        *types.Package
	CurrentVersion string
}

func (r *Renderer) renderPackageFile(pkg *types.Package, filePath string, writer io.Writer) error {
	packageVersions := pkg.FileReader.Versions()

	pathAndVersion := strings.SplitN(filePath, "@", 2)
	filePath = pathAndVersion[0]

	version := packageVersions[0]

	if len(pathAndVersion) == 2 {
		version = pathAndVersion[1]
	}

	if filePath == "" {
		filePath = "README.md"
	}

	data := packageTemplateData{
		layoutTemplateData: layoutTemplateData{
			CurrentFile: filePath,
		},
		Package:        pkg,
		CurrentVersion: version,
	}

	if content, err := pkg.FileReader.ReadFile(filePath, version); err != nil {
		return fmt.Errorf("error reading file '%v' of version '%v': %w", filePath, version, err)
	} else {
		err = renderContent(string(content), filePath, &data.layoutTemplateData)
		if err != nil {
			return err
		}

		return r.executeTemplate(writer, "package.tmpl", data)
	}
}
