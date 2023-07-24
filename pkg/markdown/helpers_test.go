package markdown_test

import (
	"testing"

	"github.com/anexia-it/go.anx.io/pkg/markdown"
)

func TestExtractFirstHeader(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		label    string
		markdown string
		expected string
	}{
		{"no header", "foo baar baz", ""},
		{"plain header", "# foo bar baz", "foo bar baz"},
		{"inline code header", "# `foo bar baz`", "foo bar baz"},
		{"combined inline code and planheader", "# `foo bar` baz", "foo bar baz"},
	}

	for _, c := range testCases {
		testCase := c
		t.Run(testCase.label, func(t *testing.T) {
			t.Parallel()

			actual := markdown.ExtractFirstHeader(testCase.markdown)
			if actual != testCase.expected {
				t.Errorf("%q (actual) did not match %q (expected)", actual, testCase.expected)
			}
		})
	}
}
