package render

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func RemoveGitRepoSuffix(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}

func renderContent(content string, filePath string, data *layoutTemplateData) error {
	switch path.Ext(filePath) {
	case ".md":
		data.MarkdownContent = string(content)
	case ".go":
		data.MarkdownContent = fmt.Sprintf("# `%v`\n\n```go\n%v\n```", filePath, string(content))
	default:
		return fmt.Errorf("%w: unknown file extension", os.ErrInvalid)
	}

	return nil
}
