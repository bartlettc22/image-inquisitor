package utils

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
)

// LatestSemanticVersion finds the latest semantic version from a list of tags
func LatestSemanticVersion(versions []*registries.Tag) (*registries.Tag, error) {
	var latestSemver *semver.Version
	var latest *registries.Tag

	for _, v := range versions {
		// Parse each version string into a semver.Version
		version, err := semver.NewVersion(v.Tag)
		if err != nil {
			continue
		}

		// Must have 3 places
		// quay.io for example has tags like '608111629' which will evaluate to '608111629.0.0'
		if len(strings.Split(v.Tag, ".")) != 3 {
			continue
		}

		// Skip pre-releases
		if version.Prerelease() != "" {
			continue
		}

		// Compare and update the latest version
		if latestSemver == nil || version.GreaterThan(latestSemver) {
			latestSemver = version
			latest = v
		}
	}

	if latestSemver == nil {
		return nil, fmt.Errorf("no valid semantic versions found")
	}

	return latest, nil
}
