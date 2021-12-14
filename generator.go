package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

var mainTemplate *template.Template
var packageTemplate *template.Template

type templateDataContent struct {
	Markdown string
}

type mainTemplateData struct {
	Title       string
	Content     templateDataContent
	CurrentFile string
	Packages    []*pkgEntry
}

type packageTemplateData struct {
	mainTemplateData

	Package        *pkgEntry
	CurrentVersion string
}

func loadTemplates(templateDirPath string) error {
	templateBase := template.New("").Funcs(template.FuncMap{
		"renderMarkdown":      renderMarkdown,
		"extractFirstHeader":  extractFirstHeader,
		"removeGitRepoSuffix": removeGitRepoSuffix,
	})

	var err error
	mainTemplate, err = templateBase.Clone()
	if err != nil {
		return fmt.Errorf("error cloning base template for main template: %w", err)
	}

	_, err = mainTemplate.ParseFiles(path.Join(templateDirPath, "layout.tmpl"))
	if err != nil {
		return fmt.Errorf("error parsing templates: %w", err)
	}

	packageTemplate, err = mainTemplate.Clone()
	if err != nil {
		return fmt.Errorf("error cloning main template for package template: %w", err)
	}

	_, err = packageTemplate.ParseFiles(path.Join(templateDirPath, "package.tmpl"))
	if err != nil {
		return fmt.Errorf("error parsing templates: %w", err)
	}

	return nil
}

func renderFiles(destinationPath string) error {
	for _, pkg := range pkgConfig {
		err := renderPackageFiles(destinationPath, pkg)
		if err != nil {
			return err
		}
	}

	content := os.DirFS("content")
	mdFiles, err := fs.Glob(content, "*.md")
	if err != nil {
		return err
	}

	mdFiles = append(mdFiles, "")

	for _, f := range mdFiles {
		destinationDirectory := path.Join(destinationPath, f)
		err := os.MkdirAll(destinationDirectory, 0755)
		if err != nil {
			return err
		}

		w, err := os.OpenFile(path.Join(destinationDirectory, "index.html"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		err = renderFile(nil, path.Join("content", f), w)
		if err != nil {
			return err
		}
	}

	return nil
}

func renderPackageFiles(destinationPath string, pkg *pkgEntry) error {
	pkgPath := path.Join(destinationPath, pkg.TargetName)
	if err := os.MkdirAll(pkgPath, os.ModeDir|0755); err != nil {
		return err
	}

	fileList := make([]string, 0, len(pkg.Versions)+2)

	// generate index and README.md for latest version
	fileList = append(fileList, "", "README.md")

	for _, v := range pkg.Versions {
		fileList = append(fileList, fmt.Sprintf("%s@%s", "README.md", v))
	}

	for _, f := range fileList {
		destinationDirectory := path.Join(pkgPath, f)
		err := os.MkdirAll(destinationDirectory, 0755)
		if err != nil {
			return err
		}

		w, err := os.OpenFile(path.Join(destinationDirectory, "index.html"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		err = renderFile(pkg, f, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func renderFile(pkg *pkgEntry, filePath string, writer io.Writer) error {
	if pkg != nil {
		return renderPackageFile(pkg, filePath, writer)
	}

	data := mainTemplateData{
		CurrentFile: filePath,
		Packages:    pkgConfig,
	}

	if filePath == "" || filePath == "content" {
		filePath = "content/index.md"
	} else if !strings.HasPrefix(filePath, "content/") {
		return fmt.Errorf("invalid file '%v' requested: %w", filePath, os.ErrInvalid)
	}

	if content, err := os.ReadFile(filePath); err != nil {
		return fmt.Errorf("error reading source file '%v': %w", filePath, err)
	} else {
		err = renderContent(string(content), filePath, &data)
		if err != nil {
			return err
		}

		return mainTemplate.ExecuteTemplate(writer, "main", data)
	}
}

func renderPackageFile(pkg *pkgEntry, filePath string, writer io.Writer) error {
	pathAndVersion := strings.SplitN(filePath, "@", 2)
	filePath = pathAndVersion[0]

	version := pkg.Versions[0]

	if len(pathAndVersion) == 2 {
		version = pathAndVersion[1]
	}

	if filePath == "" || filePath == "index" {
		filePath = "README.md"
	}

	data := packageTemplateData{
		mainTemplateData: mainTemplateData{
			CurrentFile: filePath,
		},
		Package:        pkg,
		CurrentVersion: version,
	}

	if content, err := pkg.FileReader.ReadFile(filePath, version); err != nil {
		return err
	} else {
		err = renderContent(string(content), filePath, &data.mainTemplateData)
		if err != nil {
			return err
		}

		return packageTemplate.ExecuteTemplate(writer, "main", data)
	}
}

func renderContent(content string, filePath string, data *mainTemplateData) error {
	switch path.Ext(filePath) {
	case ".md":
		data.Content.Markdown = string(content)
	case ".go":
		data.Content.Markdown = fmt.Sprintf("# `%v`\n\n```go\n%v\n```", filePath, string(content))
	default:
		return fmt.Errorf("%w: unknown file extension", os.ErrInvalid)
	}

	return nil
}
