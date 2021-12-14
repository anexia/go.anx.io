package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func renderMarkdown(contents string) (template.HTML, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	buffer := bytes.Buffer{}
	if err := md.Convert([]byte(contents), &buffer); err != nil {
		return "", fmt.Errorf("error processing markdown file to html: %w", err)
	}

	return template.HTML(buffer.Bytes()), nil
}

func extractFirstHeader(contents string) (string, error) {
	doc := goldmark.DefaultParser().Parse(text.NewReader([]byte(contents)))

	n := doc.FirstChild()
	for n != nil {
		if n.Kind() == ast.KindHeading {
			if n.(*ast.Heading).Level == 1 {
				if text, ok := n.FirstChild().(*ast.Text); ok {
					return string(text.Text([]byte(contents))), nil
				} else if code, ok := n.FirstChild().(*ast.CodeSpan); ok {
					return string(code.Text([]byte(contents))), nil
				}
			}
		}

		n = n.NextSibling()
	}

	return "", fmt.Errorf("%w: no heading found", os.ErrNotExist)
}

func removeGitRepoSuffix(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}
