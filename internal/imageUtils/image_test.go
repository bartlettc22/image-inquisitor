package imageUtils

import (
	"strings"
	"testing"
)

func TestParseImage(t *testing.T) {

	testCases := []struct {
		image                      string
		expectErrorContains        string
		expected                   Image
		expectedFullName           string
		expectedFullNameWithDigest string
	}{
		{"", "image could not be parsed", Image{}, "", ""},
		{"nginx", "", Image{
			Registry:   "docker.io",
			Owner:      "library",
			Repository: "nginx",
			Tag:        "latest",
			Digest:     "",
		}, "docker.io/library/nginx:latest", "docker.io/library/nginx:latest"},
		{"nginx:2", "", Image{
			Registry:   "docker.io",
			Owner:      "library",
			Repository: "nginx",
			Tag:        "2",
			Digest:     "",
		}, "docker.io/library/nginx:2", "docker.io/library/nginx:2"},
		{"nginx@sha256:ae123d7b8c3e7f9b9b2e0ed4b91b9a760d3cda0a6f874d81b5693b4de6d2d398", "", Image{
			Registry:   "docker.io",
			Owner:      "library",
			Repository: "nginx",
			Tag:        "latest",
			Digest:     "sha256:ae123d7b8c3e7f9b9b2e0ed4b91b9a760d3cda0a6f874d81b5693b4de6d2d398",
		}, "docker.io/library/nginx:latest", "docker.io/library/nginx:latest@sha256:ae123d7b8c3e7f9b9b2e0ed4b91b9a760d3cda0a6f874d81b5693b4de6d2d398"},
		{"docker.io/library/nginx:v2@sha256:ae123d7b8c3e7f9b9b2e0ed4b91b9a760d3cda0a6f874d81b5693b4de6d2d398", "", Image{
			Registry:   "docker.io",
			Owner:      "library",
			Repository: "nginx",
			Tag:        "v2",
			Digest:     "sha256:ae123d7b8c3e7f9b9b2e0ed4b91b9a760d3cda0a6f874d81b5693b4de6d2d398",
		}, "docker.io/library/nginx:v2", "docker.io/library/nginx:v2@sha256:ae123d7b8c3e7f9b9b2e0ed4b91b9a760d3cda0a6f874d81b5693b4de6d2d398"},
		{"foo/bar", "", Image{
			Registry:   "docker.io",
			Owner:      "foo",
			Repository: "bar",
			Tag:        "latest",
			Digest:     "",
		}, "docker.io/foo/bar:latest", "docker.io/foo/bar:latest"},
		{"foo/bar:v2", "", Image{
			Registry:   "docker.io",
			Owner:      "foo",
			Repository: "bar",
			Tag:        "v2",
			Digest:     "",
		}, "docker.io/foo/bar:v2", "docker.io/foo/bar:v2"},
		{"a/foo/bar", "non-domain as first delimiter", Image{}, "", ""},
		{"a/foo/bar:v2", "non-domain as first delimiter", Image{}, "", ""},
		{"a.com/foo/bar", "", Image{
			Registry:   "a.com",
			Owner:      "foo",
			Repository: "bar",
			Tag:        "latest",
			Digest:     "",
		}, "a.com/foo/bar:latest", "a.com/foo/bar:latest"},
		{"a.io/foo/bar:abc", "", Image{
			Registry:   "a.io",
			Owner:      "foo",
			Repository: "bar",
			Tag:        "abc",
			Digest:     "",
		}, "a.io/foo/bar:abc", "a.io/foo/bar:abc"},
		{"b.net/foo/bar/fizz/buzz:abc", "", Image{
			Registry:   "b.net",
			Owner:      "foo/bar/fizz",
			Repository: "buzz",
			Tag:        "abc",
			Digest:     "",
		}, "b.net/foo/bar/fizz/buzz:abc", "b.net/foo/bar/fizz/buzz:abc"},
	}

	for _, tc := range testCases {
		result, err := ParseImage(tc.image)
		if tc.expectErrorContains != "" && err == nil {
			t.Errorf("expected error for image `%s` but got none; want %s", tc.image, tc.expectErrorContains)
		} else if tc.expectErrorContains == "" && err != nil {
			t.Errorf("did not expect error for image `%s`; got `%s`", tc.image, err.Error())
		} else if tc.expectErrorContains != "" && err != nil {
			if !strings.Contains(err.Error(), tc.expectErrorContains) {
				t.Errorf("error did not contain expected string for image `%s`; want `%s`, got `%s`", tc.image, tc.expectErrorContains, err.Error())
			}
		} else {
			if result.Image != tc.image {
				t.Errorf("image for `%s` not correct; want `%s`, got `%s`", tc.image, tc.image, result.Image)
			}
			if result.Registry != tc.expected.Registry {
				t.Errorf("registry for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expected.Registry, result.Registry)
			}
			if result.Owner != tc.expected.Owner {
				t.Errorf("owner for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expected.Owner, result.Owner)
			}
			if result.Repository != tc.expected.Repository {
				t.Errorf("repository for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expected.Repository, result.Repository)
			}
			if result.Tag != tc.expected.Tag {
				t.Errorf("tag for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expected.Tag, result.Tag)
			}
			if result.Digest != tc.expected.Digest {
				t.Errorf("digest for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expected.Digest, result.Digest)
			}
			if fullname := result.FullyQualifiedName(false); fullname != tc.expectedFullName {
				t.Errorf("Full name for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expectedFullName, fullname)
			}
			if fullname := result.FullyQualifiedName(true); fullname != tc.expectedFullNameWithDigest {
				t.Errorf("Full name (with digest) for `%s` not correct; want `%s`, got `%s`", tc.image, tc.expectedFullNameWithDigest, fullname)
			}
		}
	}
}
