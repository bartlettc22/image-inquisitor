package utils

import (
	"testing"

	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLatestSemanticVersion(t *testing.T) {

	testCases := []struct {
		tagList       []*registries.Tag
		expectedTag   *registries.Tag
		expectedError string
	}{
		// Vanilla without "v"
		{
			[]*registries.Tag{
				{
					Tag: "1.2.3",
				},
				{
					Tag: "1.3.3",
				},
				{
					Tag: "latest",
				},
			},
			&registries.Tag{
				Tag: "1.3.3",
			},
			"",
		},
		// Vanilla with "v"
		{
			[]*registries.Tag{
				{
					Tag: "v1.2.3",
				},
				{
					Tag: "v1.3.3",
				},
				{
					Tag: "latest",
				},
			},
			&registries.Tag{
				Tag: "v1.3.3",
			},
			"",
		},
		// Ensure that versions without 3 "." aren't automatically expanded
		// i.e. 608111629 does not count as 608111629.0.0
		{
			[]*registries.Tag{
				{
					Tag: "608111629",
				},
				{
					Tag: "v1.3.3",
				},
				{
					Tag: "latest",
				},
			},
			&registries.Tag{
				Tag: "v1.3.3",
			},
			"",
		},
		// Ensure we don't count pre-release
		{
			[]*registries.Tag{
				{
					Tag: "608111629",
				},
				{
					Tag: "1.2.3-beta",
				},
				{
					Tag: "latest",
				},
			},
			nil,
			"no valid semantic versions found",
		},
	}

	for _, tc := range testCases {
		actualTag, err := LatestSemanticVersion(tc.tagList)
		if tc.expectedError == "" {
			require.NoError(t, err)
			assert.Equal(t, tc.expectedTag.Tag, actualTag.Tag)
		} else {
			require.ErrorContains(t, err, tc.expectedError)
		}
	}
}
