package utils

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// LatestSemanticVersionStr returns the latest semantic version string
// from a list of strings. If no versions are found, returns an empty string.
//
// Requires that the version strings are semver-compatible and have 3 "." characters.
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

		// Must have 3 places
		// quay.io for example has tags like '608111629' which will evaluate to '608111629.0.0'
		if len(strings.Split(v, ".")) != 3 {
			continue
		}

		// Skip pre-releases
		if version.Prerelease() != "" {
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
