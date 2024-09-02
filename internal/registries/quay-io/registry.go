package quay_io

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
	RegistryHost = "quay.io"
)

type Tag struct {
	Name           string `json:"name"`
	StartTimestamp int    `json:"start_ts"`
}

// TagsResponse represents the response from Quay.io API
type TagsResponse struct {
	Tags          []Tag `json:"tags"`
	Page          int   `json:"page"`
	HasAdditional bool  `json:"has_additional"`
}

type QuayIORegistry struct{}

func NewRegistry() *QuayIORegistry {
	return &QuayIORegistry{}
}

func (r *QuayIORegistry) IsRegistry(registry string) bool {
	return registry == RegistryHost
}

func (r *QuayIORegistry) FetchReport(image *imageUtils.Image) (*registries.ImageReport, error) {
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
		CurrentTagAge:       time.Since(currentTagsResponse[0].TagTimestamp),
		LatestTag:           latest.Tag,
		LatestTagTimestamp:  latest.TagTimestamp,
		LatestTagAge:        time.Since(latest.TagTimestamp),
	}, nil
}

func FetchLatestSemanticVersion(namespace, repository string) (*registries.Tag, error) {
	tags, err := FetchTags(namespace, repository, "")
	if err != nil {
		return nil, err
	}

	return utils.LatestSemanticVersion(tags)
}

// FetchTags retrieves tags from Quay.io API
func FetchTags(namespace, repository, tag string) ([]*registries.Tag, error) {
	page := 1

	var tags []*registries.Tag

	for {
		url := fmt.Sprintf("https://quay.io/api/v1/repository/%s/%s/tag/?page=%d", namespace, repository, page)
		if tag != "" {
			url = url + fmt.Sprintf("&specificTag=%s", tag)
		}
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get tags: %s", resp.Status)
		}

		var result TagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, tag := range result.Tags {
			ts := time.Unix(int64(tag.StartTimestamp), 0)
			tags = append(tags, &registries.Tag{
				Tag:          tag.Name,
				TagTimestamp: ts,
			})
		}

		if !result.HasAdditional {
			break
		}
		page++
	}
	return tags, nil
}
