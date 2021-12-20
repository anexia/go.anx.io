package render

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"time"

	"html/template"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
)

type layoutTemplateData struct {
	Title           string
	CurrentFile     string
	MarkdownContent string
}

type commonTemplateData struct {
	CurrentTime time.Time
	PageData    interface{}

	Version   string
	SourceURL string
}

func loadTemplates(templatePath string) (map[string]*template.Template, error) {
	baseTemplate, err := template.New("").Funcs(template.FuncMap{
		"formatDate":          formatDate,
		"renderMarkdown":      markdown.RenderMarkdown,
		"extractFirstHeader":  markdown.ExtractFirstHeader,
		"removeGitRepoSuffix": RemoveGitRepoSuffix,
		"default": func(d string, v string) string {
			if v == "" {
				return d
			}

			return v
		},
	}).ParseFiles(path.Join(templatePath, "layout.tmpl"))

	if err != nil {
		return nil, fmt.Errorf("error parsing layout template: %w", err)
	}

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

func (r *Renderer) executeTemplate(w io.Writer, name string, data interface{}) error {
	tmpl, ok := r.templates[name]
	if !ok {
		return &template.Error{
			ErrorCode: template.ErrNoSuchTemplate,
		}
	}

	return tmpl.Execute(w, commonTemplateData{
		CurrentTime: time.Now(),
		Version:     r.version,
		SourceURL:   r.sourceURL,
		PageData:    data,
	})
}
