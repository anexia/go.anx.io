package render

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
	"github.com/anexia-it/go.anx.io/pkg/types"
)

type mainTemplateData struct {
	layoutTemplateData
	Packages []*types.Package
}

func (r Renderer) renderContentFile(filePath string, writer io.Writer) error {

	if filePath == "" {
		filePath = "index.md"
	} else if filePath == "chroma/style.css" {
		return markdown.RenderCodeCSS(writer)
	}

	if content, err := os.ReadFile(path.Join("content", filePath)); err != nil {
		return fmt.Errorf("error reading source file '%v': %w", filePath, err)
	} else {
		data := mainTemplateData{
			layoutTemplateData: layoutTemplateData{
				CurrentYear: time.Now().Year(),
				CurrentFile: filePath,
			},
			Packages: r.packages,
		}

		err = renderContent(string(content), filePath, &data.layoutTemplateData)
		if err != nil {
			return err
		}

		return r.templates["main.tmpl"].Execute(writer, data)
	}
}
