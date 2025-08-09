package registries

import (
	"github.com/bartlettc22/image-inquisitor/internal/utils"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func FetchLatestSemverTagByStr(repoStr string) (string, error) {
	repository, err := name.NewRepository(repoStr)
	if err != nil {
		return "", err
	}

	tags, err := remote.List(repository, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", err
	}

	return utils.LatestSemanticVersionStr(tags), nil
}
