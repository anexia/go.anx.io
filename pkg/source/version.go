package source

import (
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func sortVersions(versions []string) {
	//nolint:varnamelen // what else to call a and b here?!
	sort.Slice(versions, func(a int, b int) bool {
		verA, _ := semver.NewVersion(versions[a])
		verB, _ := semver.NewVersion(versions[b])

		switch {
		case verA != nil && verB != nil:
			return verA.Compare(verB) > 0
		case verA != nil && verB == nil:
			return true
		case verA == nil && verB != nil:
			return false
		default:
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
		}
	})
}
