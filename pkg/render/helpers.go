package render

import (
	"strings"
	"time"
)

func RemoveGitRepoSuffix(repo string) string {
	return strings.TrimSuffix(repo, ".git")
}

func formatDate(format string, t time.Time) string {
	return t.Format(format)
}
