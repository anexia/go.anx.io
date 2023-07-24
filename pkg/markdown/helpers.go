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

	markdown := goldmark.New(
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
	if err := markdown.Convert([]byte(contents), &buffer); err != nil {
		return "", fmt.Errorf("error processing markdown file to html: %w", err)
	}

	// #nosec
	return template.HTML(buffer.Bytes()), nil
}

func ExtractFirstHeader(contents string) string {
	doc := goldmark.DefaultParser().Parse(text.NewReader([]byte(contents)))

	potentialFirstHeader := doc.FirstChild()
	for potentialFirstHeader != nil {
		if potentialFirstHeader.Kind() == ast.KindHeading {
			//nolint:forcetypeassert // already checked above
			if potentialFirstHeader.(*ast.Heading).Level == 1 {
				if text, ok := potentialFirstHeader.FirstChild().(*ast.Text); ok {
					return string(text.Text([]byte(contents)))
				} else if code, ok := potentialFirstHeader.FirstChild().(*ast.CodeSpan); ok {
					return string(code.Text([]byte(contents)))
				}
			}
		}

		potentialFirstHeader = potentialFirstHeader.NextSibling()
	}

	return ""
}
