package source

import (
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func sortVersions(versions []string) {
	sort.Slice(versions, func(a int, b int) bool {
		verA, errA := semver.NewVersion(versions[a])
		verB, errB := semver.NewVersion(versions[b])

		if errA == nil && errB != nil {
			return true
		} else if errB == nil && errA != nil {
			return false
		} else if verA == verB && verA == nil {
			if versions[a] == "main" {
				return true
			} else if versions[b] == "main" {
				return false
			} else if versions[a] == "master" {
				return false
			}

			return strings.Compare(versions[a], versions[b]) > 0
		} else {
			return verA.Compare(verB) > 0
		}
	})
}
