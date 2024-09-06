package ghcr

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/utils"
)

const (
	registryHost = "ghcr.io"
)

type tokenResponse struct {
	Token string
}

type tagListResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

type manifestIndexResponse struct {
	MediaType string
	Manifests []*manifestResponseManifest
	Config    *manifestResponseConfig
}

type manifestResponseManifest struct {
	MediaType string
	Digest    string
	Platform  *manifestResponsePlatform
}

type manifestResponsePlatform struct {
	Architecture string
	OS           string
}

type manifestResponse struct {
	Config *manifestResponseConfig
}

type manifestResponseConfig struct {
	Digest string
}

type manifestConfigResponse struct {
	Created time.Time
}

type GHCRRegistry struct {
	tokenCache sync.Map
}

func NewRegistry() *GHCRRegistry {
	return &GHCRRegistry{}
}

func (r *GHCRRegistry) IsRegistry(registry string) bool {
	return registry == registryHost
}

func (r *GHCRRegistry) FetchReport(image *imageUtils.Image) (*registries.RegistryImageReport, error) {
	report := &registries.RegistryImageReport{
		Tag: image.Tag,
	}

	currentTagsResponse, err := r.getTag(image.Owner, image.Repository, image.Tag)
	if err != nil {
		return report, fmt.Errorf("ghcr.io image %s/%s: %v", image.Owner, image.Repository, err)
	}
	if currentTagsResponse.TagTimestamp.IsZero() {
		return report, fmt.Errorf("could not find current tag: %s", image.Tag)
	}
	report.TagTimestamp = currentTagsResponse.TagTimestamp

	// Fetch latest Tag
	latest, err := r.fetchLatestSemanticVersion(image.Owner, image.Repository)
	if err != nil {
		return report, fmt.Errorf("ghcr.io image %s/%s: %v", image.Owner, image.Repository, err)
	}
	report.LatestTag = latest.Tag
	report.LatestTagTimestamp = latest.TagTimestamp

	return report, nil
}

func (r *GHCRRegistry) fetchLatestSemanticVersion(owner, repository string) (*registries.Tag, error) {
	url := fmt.Sprintf("https://ghcr.io/v2/%s/%s/tags/list", owner, repository)
	tmpTags := []*registries.Tag{}

	for url != "" {
		var bodyBytes []byte
		var err error
		bodyBytes, url, err = r.get(owner, repository, url, []string{})
		if err != nil {
			return nil, err
		}

		tagListResponse := &tagListResponse{}
		err = json.Unmarshal(bodyBytes, tagListResponse)
		if err != nil {
			return nil, err
		}

		for _, tag := range tagListResponse.Tags {
			tmpTags = append(tmpTags, &registries.Tag{
				Tag: tag,
			})
		}
	}

	latestTag, err := utils.LatestSemanticVersion(tmpTags)
	if err != nil {
		return nil, err
	}

	return r.getTag(owner, repository, latestTag.Tag)
}

func (r *GHCRRegistry) getTag(owner, repository, tag string) (*registries.Tag, error) {

	url := fmt.Sprintf("https://ghcr.io/v2/%s/%s/manifests/%s", owner, repository, tag)
	bodyBytes, _, err := r.get(owner, repository, url, []string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
	})
	if err != nil {
		return nil, err
	}

	manifestIndexResponse := &manifestIndexResponse{}
	err = json.Unmarshal(bodyBytes, manifestIndexResponse)
	if err != nil {
		return nil, err
	}

	configDigest := ""
	if manifestIndexResponse.MediaType == "application/vnd.oci.image.index.v1+json" {
		// Newer images respond with an index of manifests (arm, amd64, etc.)
		// So an additional query is required to get the config
		var desiredManifest *manifestResponseManifest
		for _, manifest := range manifestIndexResponse.Manifests {
			if manifest.Platform.Architecture == "amd64" {
				desiredManifest = manifest
				break
			}
		}
		if desiredManifest == nil {
			return nil, fmt.Errorf("unable to find github manifest at: '%s'", url)
		}

		url = fmt.Sprintf("https://ghcr.io/v2/%s/%s/manifests/%s", owner, repository, desiredManifest.Digest)
		bodyBytes, _, err = r.get(owner, repository, url, []string{
			"application/vnd.oci.image.manifest.v1+json",
		})
		if err != nil {
			return nil, err
		}

		manifestResponse := &manifestResponse{}
		err = json.Unmarshal(bodyBytes, manifestResponse)
		if err != nil {
			return nil, err
		}

		configDigest = manifestResponse.Config.Digest

	} else if manifestIndexResponse.MediaType == "application/vnd.docker.distribution.manifest.v2+json" {
		// Older images respond with the manifest directly
		configDigest = manifestIndexResponse.Config.Digest
	} else {
		return nil, fmt.Errorf("unknown index media type in response '%s': %v", url, manifestIndexResponse.MediaType)
	}

	manifestConfigResponse, err := r.getManifestConfig(owner, repository, configDigest)
	if err != nil {
		return nil, err
	}

	return &registries.Tag{
		Tag:          tag,
		TagTimestamp: manifestConfigResponse.Created,
	}, nil
}

func (r *GHCRRegistry) getManifestConfig(owner, repository, digest string) (*manifestConfigResponse, error) {
	url := fmt.Sprintf("https://ghcr.io/v2/%s/%s/blobs/%s", owner, repository, digest)
	bodyBytes, _, err := r.get(owner, repository, url, []string{"application/vnd.oci.image.config.v1+json"})
	if err != nil {
		return nil, err
	}

	manifestConfigResponse := &manifestConfigResponse{}
	err = json.Unmarshal(bodyBytes, manifestConfigResponse)
	if err != nil {
		return nil, err
	}

	return manifestConfigResponse, nil
}

func (r *GHCRRegistry) get(owner, repository, url string, acceptHeaders []string) ([]byte, string, error) {

	userImage := fmt.Sprintf("%s/%s", owner, repository)
	token, ok := r.tokenCache.Load(userImage)
	if !ok {
		tokenURL := fmt.Sprintf("https://ghcr.io/token?scope=repository:%s:pull", userImage)
		resp, err := http.Get(tokenURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to get token at '%s': %v", tokenURL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, "", fmt.Errorf("failed to get token at '%s': %s", tokenURL, resp.Status)
		}

		var tokenResponse tokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
			return nil, "", fmt.Errorf("failed to decode body at '%s': %v", tokenURL, err)
		}
		r.tokenCache.Store(userImage, tokenResponse.Token)
		token = tokenResponse.Token
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request at '%s': %v", url, err)
	}

	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	for _, acceptHeader := range acceptHeaders {
		request.Header.Set("Accept", acceptHeader)
	}

	client := &http.Client{}

	resp, err := client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get request at '%s': %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to get request at '%s': %s", url, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read body for '%s': %s", url, err)
	}

	linkHeader := resp.Header.Get("Link")
	nextLink := ""
	if linkHeader != "" {
		nextLinkPath := parseLinkHeaderRelNext(linkHeader)
		if nextLinkPath != "" {
			nextLink = "https://ghcr.io" + nextLinkPath
		}
	}

	return bodyBytes, nextLink, nil

}

func parseLinkHeaderRelNext(linkHeader string) string {

	// Split the header by commas to separate the individual links
	partsIndividualLinks := strings.Split(linkHeader, ",")

	for _, linkLine := range partsIndividualLinks {
		linkLineParts := strings.Split(linkLine, ";")
		if len(linkLineParts) >= 2 {
			// First part is the URL path, surrounded by <>
			// Second+ parts are the attributes

			// Trim the URL path
			url := strings.Trim(linkLineParts[0], "<")
			url = strings.Trim(url, ">")

			// Trim the "rel"
			for _, part := range linkLineParts[1:] {
				fullAttr := strings.TrimSpace(part)
				attrParts := strings.Split(fullAttr, "=")

				// First part is the key, second part is the quoted value
				if len(attrParts) == 2 {
					attrKey := attrParts[0]
					attrValue := strings.Trim(attrParts[1], "\"")
					if attrKey == "rel" && attrValue == "next" {
						return url
					}
				}
			}
		}
	}

	return ""
}
