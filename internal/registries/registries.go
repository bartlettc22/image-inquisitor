package registries

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Image struct {
	ref          name.Reference
	architecture string
	os           string
	image        v1.Image
}

func NewImage(imageName string) (*Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err
	}
	return &Image{
		ref: ref,
		// Hardcoded for now
		architecture: "amd64",
		// Hardcoded for now
		os: "linux",
	}, nil
}

// Name returns the original given name of the image
func (i *Image) Name() string {
	return i.ref.String()
}

func (i *Image) Registry() string {
	return i.ref.Context().RegistryStr()
}

func (i *Image) Repository() string {
	return i.ref.Context().RepositoryStr()
}

// ExpandedRepository returns a normalized name, with registry and
// without the tag/digest, that can be compared for equality.
// For references to shortened docker.io images (i.e. "nginx:latest"),
// this will be the full name with registry (i.e. "index.docker.io/library/nginx")
func (i *Image) ExpandedRepository() string {
	return i.ref.Context().String()
}

func (i *Image) Digest() (string, error) {
	image, err := i.remoteImage()
	if err != nil {
		return "", err
	}

	digest, err := image.Digest()
	if err != nil {
		return "", err
	}

	return digest.String(), nil
}

func (i *Image) Architecture() (string, error) {
	image, err := i.remoteImage()
	if err != nil {
		return "", err
	}

	config, err := image.ConfigFile()
	if err != nil {
		return "", err
	}

	return config.Architecture, nil
}

func (i *Image) OS() (string, error) {
	image, err := i.remoteImage()
	if err != nil {
		return "", err
	}

	config, err := image.ConfigFile()
	if err != nil {
		return "", err
	}
	return config.OS, nil
}

func (i *Image) remoteImage() (v1.Image, error) {

	if i.image != nil {
		return i.image, nil
	}
	image, err := remote.Image(i.ref,
		remote.WithPlatform(v1.Platform{
			Architecture: i.architecture,
			OS:           i.os,
		}),
		remote.WithAuthFromKeychain(authn.DefaultKeychain),
	)
	if err != nil {
		return nil, err
	}
	i.image = image
	return i.image, nil
}
