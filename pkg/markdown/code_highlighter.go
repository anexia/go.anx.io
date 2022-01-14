package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

var chromaStyle = styles.Get("pygments")

var chromaFormatterOpts = []html.Option{
	html.Standalone(false),
	html.WithAllClasses(true),
	html.WithClasses(true),
	html.TabWidth(4),
	html.WithLineNumbers(true),
}

// codeHighlighterImpl implements an extension for goldmark to make code blocks with
// syntax highlighting. We are not using github.com/yuin/goldmark-highlighting to
// enable links to line numbers (we have to count code blocks for that).
// Also I wrote this before finding the existing extension ...
//  -- Mara @LittleFox94 Grosch, 2021-12-15
type codeHighlighterImpl struct {
	codeIDCounter int
}

func codeHighlighter() *codeHighlighterImpl {
	return &codeHighlighterImpl{
		codeIDCounter: 1,
	}
}

func (ch *codeHighlighterImpl) Extend(md goldmark.Markdown) {
	md.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.PrioritizedValue{
				Value:    ch,
				Priority: 0,
			},
		),
	)
}

func (ch *codeHighlighterImpl) RegisterFuncs(nrfr renderer.NodeRendererFuncRegisterer) {
	nrfr.Register(ast.KindCodeBlock, ch.renderHighlightedCode)
	nrfr.Register(ast.KindFencedCodeBlock, ch.renderHighlightedCode)
}

func (ch *codeHighlighterImpl) renderHighlightedCode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	language := ""
	if fcb, ok := node.(*ast.FencedCodeBlock); ok {
		language = string(fcb.Language(source))
	}

	code := strings.Builder{}

	for i := 0; i < node.Lines().Len(); i++ {
		line := node.Lines().At(i)
		code.Write(line.Value(source))
	}

	var lexer chroma.Lexer

	if language == "" {
		lexer = lexers.Analyse(code.String())
	} else {
		lexer = lexers.Get(language)
	}

	if lexer == nil {
		lexer = lexers.Fallback
	}

	tokens, err := lexer.Tokenise(nil, code.String())
	if err != nil {
		return ast.WalkContinue, err //nolint:wrapcheck
	}

	codeLinkID := fmt.Sprintf("code-%v-", ch.codeIDCounter)
	ch.codeIDCounter++

	formatter := html.New(
		append(chromaFormatterOpts, html.LinkableLineNumbers(true, codeLinkID))...,
	)

	return ast.WalkContinue, formatter.Format(w, chromaStyle, tokens)
}

func RenderCodeCSS(w io.Writer) error {
	formatter := html.New(chromaFormatterOpts...)

	return formatter.WriteCSS(w, chromaStyle)
}
