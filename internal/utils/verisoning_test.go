package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatestSemanticVersion(t *testing.T) {

	testCases := []struct {
		tagList     []string
		expectedTag string
	}{
		// Vanilla without "v"
		{
			[]string{
				"1.2.3",
				"1.3.3",
				"latest",
			},
			"1.3.3",
		},
		// Vanilla with "v"
		{
			[]string{
				"v1.2.3",
				"v1.3.3",
				"latest",
			},
			"v1.3.3",
		},
		// Ensure that versions without 3 "." aren't automatically expanded
		// i.e. 608111629 does not count as 608111629.0.0
		{
			[]string{
				"608111629",
				"v1.3.3",
				"latest",
			},
			"v1.3.3",
		},
		// Ensure we don't count pre-release
		{
			[]string{
				"608111629",
				"1.2.3-beta",
				"latest",
			},
			"",
		},
	}

	for _, tc := range testCases {
		actualTag := LatestSemanticVersionStr(tc.tagList)
		assert.Equal(t, tc.expectedTag, actualTag)
	}
}
