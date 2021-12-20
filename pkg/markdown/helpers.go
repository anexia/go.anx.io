package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var ErrNoHeadingFound = errors.New("no heading found")

func RenderMarkdown(contents string) (template.HTML, error) {
	highlighter := codeHighlighter()

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
			highlighter,
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

func ExtractFirstHeader(contents string) string {
	doc := goldmark.DefaultParser().Parse(text.NewReader([]byte(contents)))

	n := doc.FirstChild()
	for n != nil {
		if n.Kind() == ast.KindHeading {
			if n.(*ast.Heading).Level == 1 {
				if text, ok := n.FirstChild().(*ast.Text); ok {
					return string(text.Text([]byte(contents)))
				} else if code, ok := n.FirstChild().(*ast.CodeSpan); ok {
					return string(code.Text([]byte(contents)))
				}
			}
		}

		n = n.NextSibling()
	}

	return ""
}
