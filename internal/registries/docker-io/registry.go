package docker_io

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/utils"
)

const (
	RegistryHost = "docker.io"
)

// TagsResponse represents the response from Docker.io API
type TagsResponse struct {
	Next    string   `json:"next"`
	Results []Result `json:"results"`
}

type Result struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"last_updated"`
}

type DockerIORegistry struct{}

func NewRegistry() *DockerIORegistry {
	return &DockerIORegistry{}
}

func (r *DockerIORegistry) IsRegistry(registry string) bool {
	return registry == RegistryHost
}

func (r *DockerIORegistry) FetchReport(image *imageUtils.Image) (*registries.ImageReport, error) {
	latest, err := FetchLatestSemanticVersion(image.Owner, image.Repository)
	if err != nil {
		return nil, err
	}

	currentTagsResponse, err := FetchTags(image.Owner, image.Repository, image.Tag)
	if err != nil {
		return nil, err
	}
	if len(currentTagsResponse) != 1 {
		return nil, fmt.Errorf("could not find current tag: %s", image.Tag)
	}

	return &registries.ImageReport{
		CurrentTag:          image.Tag,
		CurrentTagTimestamp: currentTagsResponse[0].TagTimestamp,
		LatestTag:           latest.Tag,
		LatestTagTimestamp:  latest.TagTimestamp,
	}, nil
}

func FetchLatestSemanticVersion(owner, repository string) (*registries.Tag, error) {
	tags, err := FetchTags(owner, repository, "")
	if err != nil {
		return nil, err
	}

	tag, err := utils.LatestSemanticVersion(tags)
	if err != nil {
		return nil, fmt.Errorf("docker.io image %s/%s: %v", owner, repository, err)
	}

	return tag, nil
}

// FetchTags retrieves tags from Quay.io API
func FetchTags(owner, repository, tag string) ([]*registries.Tag, error) {

	var tags []*registries.Tag

	nextURL := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags/?page_size=100", owner, repository)
	if tag != "" {
		nextURL = fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags/%s/?page_size=100", owner, repository, tag)
	}
	for {
		resp, err := http.Get(nextURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get tags: %s", resp.Status)
		}

		// multi-tag response
		if tag == "" {
			var result TagsResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return nil, err
			}
			for _, result := range result.Results {
				tags = append(tags, &registries.Tag{
					Tag:          result.Name,
					TagTimestamp: result.LastUpdated,
				})
			}

			nextURL = result.Next
			if nextURL == "" {
				break
			}
			// single-tag response
		} else {
			var result Result
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return nil, err
			}
			tags = append(tags, &registries.Tag{
				Tag:          result.Name,
				TagTimestamp: result.LastUpdated,
			})
			break
		}

		// TODO: back off and retry 429
		time.Sleep(200 * time.Millisecond)
	}
	return tags, nil
}
