package utils

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// LatestSemanticVersionStr returns the latest semantic version string
// from a list of strings. If no versions are found, returns an empty string.
//
// Does not consider pre-release versions.
func LatestSemanticVersionStr(versions []string) string {
	var latestSemver *semver.Version

	for _, v := range versions {
		// Parse each version string into a semver.Version
		// Ignore errors (i.e. non-semver strings)
		version, err := semver.NewVersion(v)
		if err != nil {
			continue
		}

		// Skip pre-releases
		if version.Prerelease() != "" {
			continue
		}

		// Must have at least 2-3 places
		// For example has tags like '608111629' will not be considered
		if len(strings.Split(v, ".")) < 2 {
			continue
		}

		// Compare and update the latest version
		if latestSemver == nil || version.GreaterThan(latestSemver) {
			latestSemver = version
		}
	}

	if latestSemver == nil {
		return ""
	}

	return latestSemver.Original()
}
