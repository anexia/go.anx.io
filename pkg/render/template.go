package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
)

type layoutTemplateData struct {
	Title       string
	CurrentFile string
	CurrentYear int

	MarkdownContent string
}

func loadTemplates(templatePath string) (map[string]*template.Template, error) {
	baseTemplate, err := template.New("").Funcs(template.FuncMap{
		"renderMarkdown":      markdown.RenderMarkdown,
		"extractFirstHeader":  markdown.ExtractFirstHeader,
		"removeGitRepoSuffix": RemoveGitRepoSuffix,
	}).ParseFiles(path.Join(templatePath, "layout.tmpl"))

	files, err := fs.Glob(os.DirFS(templatePath), "*.tmpl")

	if err != nil {
		return nil, fmt.Errorf("error searching templates: %w", err)
	}

	ret := make(map[string]*template.Template, len(files)-1)

	for _, f := range files {
		if f == "layout.tmpl" {
			continue
		}

		tmpl, err := template.Must(baseTemplate.Clone()).ParseFiles(
			path.Join(templatePath, f),
		)

		if err != nil {
			return nil, fmt.Errorf("error parsing layout template: %w", err)
		}

		ret[f] = tmpl
	}

	return ret, nil
}
