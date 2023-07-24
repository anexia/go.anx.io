package source

import (
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

//nolint:varnamelen // what else to call a and b here?!
func compareSemver(a, b *semver.Version) bool {
	switch {
	case a != nil && b != nil:
		return a.Compare(b) > 0
	case a != nil && b == nil:
		return true
	case a == nil && b != nil:
		return false
	default:
		return false
	}
}

func sortVersions(versions []string) {
	//nolint:varnamelen // what else to call a and b here?!
	sort.Slice(versions, func(a int, b int) bool {
		verA, _ := semver.NewVersion(versions[a])
		verB, _ := semver.NewVersion(versions[b])

		if verA != nil || verB != nil {
			return compareSemver(verA, verB)
		}

		switch {
		case versions[a] == "main":
			return true
		case versions[b] == "main":
			return false
		case versions[a] == "master":
			return true
		case versions[b] == "master":
			return false
		default:
			return strings.Compare(versions[a], versions[b]) > 0
		}
	})
}
