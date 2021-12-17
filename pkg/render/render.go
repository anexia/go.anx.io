package render

import (
	"html/template"
	"io"

	"github.com/anexia-it/go.anx.io/pkg/types"
)

type Renderer struct {
	templates map[string]*template.Template
	packages  []*types.Package

	version   string
	sourceURL string
}

func NewRenderer(templatePath string, packages []*types.Package) (*Renderer, error) {
	templates, err := loadTemplates(templatePath)
	if err != nil {
		return nil, err
	}

	return &Renderer{
		templates: templates,
		packages:  packages,
	}, nil
}

func (r *Renderer) SetBuildInfo(version string, sourceURL string) {
	r.version = version
	r.sourceURL = sourceURL
}

func (r *Renderer) RenderFile(pkg *types.Package, filePath string, w io.Writer) error {
	if pkg == nil {
		return r.renderContentFile(filePath, w)
	} else {
		return r.renderPackageFile(pkg, filePath, w)
	}
}
