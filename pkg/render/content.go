package render

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
	"github.com/anexia-it/go.anx.io/pkg/types"
)

type mainTemplateData struct {
	layoutTemplateData
	Packages []*types.Package
}

func (r *Renderer) renderContentFile(filePath string, writer io.Writer) error {

	if filePath == "" || filePath == "index.html" {
		filePath = "index.md"
	} else if filePath == "chroma/style.css" {
		return markdown.RenderCodeCSS(writer)
	}

	if content, err := os.ReadFile(path.Join("content", filePath)); err != nil {
		return fmt.Errorf("error reading source file '%v': %w", filePath, err)
	} else {
		data := mainTemplateData{
			layoutTemplateData: layoutTemplateData{
				CurrentFile: filePath,
			},
			Packages: r.packages,
		}

		err = renderContent(string(content), filePath, &data.layoutTemplateData)
		if err != nil {
			return err
		}

		return r.executeTemplate(writer, "main.tmpl", data)
	}
}

func (r *Renderer) filesForContent() ([]string, error) {
	ret := []string{"chroma/style.css"}

	if contentFiles, err := fs.Glob(os.DirFS(r.contentPath), "*"); err != nil {
		return nil, nil
	} else {
		for _, f := range contentFiles {
			if f == "index.md" {
				f = "index.html"
			}

			ret = append(ret, f)
		}
	}

	return ret, nil
}
