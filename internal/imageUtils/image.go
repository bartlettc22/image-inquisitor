package imageUtils

import (
	"fmt"
	"path"
	"strings"
)

type ImagesList map[string]*Image

type Image struct {
	Image      string
	Registry   string
	Owner      string
	Repository string
	Tag        string
	Digest     string
}

// List lists the images as a slice of strings
func (il ImagesList) List() []string {
	list := []string{}
	for image := range il {
		list = append(list, image)
	}
	return list
}

// FullName returns the fully qualified name
func (img Image) FullName(digest bool) string {
	joinedPath := path.Join(img.Registry, img.Owner, img.Repository)
	fullName := fmt.Sprintf("%s:%s", joinedPath, img.Tag)
	if digest && img.Digest != "" {
		fullName = fmt.Sprintf("%s@%s", fullName, img.Digest)
	}
	return fullName
}

// ParseImage parses an image string into it's parts (registry, tag, etc.)
func ParseImage(image string) (*Image, error) {

	var owner, repository, tag, digest string
	registry := "docker.io"
	repoWithTag := ""

	parts := strings.Split(image, "/")

	// If length == 1, then it is an "official" docker.io image
	// i.e. nginx:latest (docker.io/library/nginx:latest)
	if len(parts) == 1 {
		// docker.io registry with "official" image
		owner = "library"
		repoWithTag = parts[0]
	} else {
		if strings.Contains(parts[0], ".") {
			registry = parts[0]
			parts = parts[1:]
		} else if len(parts) > 2 {
			return nil, fmt.Errorf("non-domain as first delimiter: '%s'", image)
		}

		owner = path.Join(parts[:len(parts)-1]...)
		repoWithTag = parts[len(parts)-1]
	}

	tagParts := strings.SplitN(repoWithTag, "@", 2)
	if len(tagParts) > 1 {
		repoWithTag = tagParts[0]
		digest = tagParts[1]
	}

	tagParts = strings.SplitN(repoWithTag, ":", 2)
	repository = tagParts[0]
	if len(tagParts) == 1 {
		tag = "latest"
	} else {
		tag = tagParts[1]
	}

	if repository == "" {
		return nil, fmt.Errorf("image could not be parsed: '%s'", image)
	}

	return &Image{
		Image:      image,
		Registry:   registry,
		Owner:      owner,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
	}, nil
}
