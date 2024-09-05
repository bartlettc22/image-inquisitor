package docker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/imageUtils"
	"github.com/bartlettc22/image-inquisitor/internal/registries"
	"github.com/bartlettc22/image-inquisitor/internal/utils"
)

const (
	registryHost = "docker.io"

	// PageSize is the max allowed page size for hub.docker.com
	pageSize = 100
)

// TagsResponse represents the response from Docker.io API
// for a list of tags
type TagsResponseList struct {
	Count   int                 `json:"count"`
	Next    string              `json:"next"`
	Results []TagResponseResult `json:"results"`
}

type TagResponseResult struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"last_updated"`
}

type DockerIORegistry struct {
	workerRequestChan chan *WorkerRequest
}

func NewRegistry() *DockerIORegistry {
	registry := &DockerIORegistry{
		workerRequestChan: make(chan *WorkerRequest),
	}
	registry.Run()
	return registry
}

func (r *DockerIORegistry) IsRegistry(registry string) bool {
	return registry == registryHost
}

func (r *DockerIORegistry) FetchReport(image *imageUtils.Image) (*registries.RegistryImageReport, error) {
	report := &registries.RegistryImageReport{
		CurrentTag: image.Tag,
	}

	currentTagsResponse, err := r.getTag(image.Owner, image.Repository, image.Tag)
	if err != nil {
		return report, fmt.Errorf("docker.io image %s/%s: %v", image.Owner, image.Repository, err)
	}
	if currentTagsResponse.TagTimestamp.IsZero() {
		return report, fmt.Errorf("could not find current tag: %s", image.Tag)
	}
	report.CurrentTagTimestamp = currentTagsResponse.TagTimestamp

	// Fetch latest Tag
	latest, err := r.fetchLatestSemanticVersion(image.Owner, image.Repository, report.CurrentTagTimestamp)
	if err != nil {
		return report, fmt.Errorf("docker.io image %s/%s: %v", image.Owner, image.Repository, err)
	}
	report.LatestTag = latest.Tag
	report.LatestTagTimestamp = latest.TagTimestamp

	return report, nil
}

func (r *DockerIORegistry) fetchLatestSemanticVersion(owner, repository string, currentTimestamp time.Time) (*registries.Tag, error) {
	tags, err := r.getAllTags(owner, repository, currentTimestamp)
	if err != nil {
		return nil, err
	}

	return utils.LatestSemanticVersion(tags)
}

func (r *DockerIORegistry) getTag(owner, repository, tag string) (*registries.Tag, error) {

	firstURL := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags/%s/", owner, repository, tag)

	responseBodyChan := make(chan *WorkerResponse, 1)
	defer close(responseBodyChan)

	r.workerRequestChan <- &WorkerRequest{
		url:          firstURL,
		responseChan: responseBodyChan,
	}

	rawResponse := <-responseBodyChan
	response := &TagResponseResult{}
	err := json.Unmarshal(rawResponse.bodyBytes, response)
	if err != nil {
		return nil, err
	}
	return &registries.Tag{
		Tag:          response.Name,
		TagTimestamp: response.LastUpdated,
	}, nil
}

func (r *DockerIORegistry) getAllTags(owner, repository string, stopAt time.Time) ([]*registries.Tag, error) {

	var tags []*registries.Tag

	responseBodyChan := make(chan *WorkerResponse, 1000)

	page := 1
	for continuing := true; continuing; {

		// Although undocumented, it seems that setting ordering=last_updated returns the results with the newest first
		// We can use this to stop searching once we've passed the last updated on the current version
		url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags/?ordering=last_updated&page_size=%d&page=%d", owner, repository, pageSize, page)
		r.workerRequestChan <- &WorkerRequest{
			url:          url,
			responseChan: responseBodyChan,
		}
		// Wait for first response
		rawResponse := <-responseBodyChan
		response := &TagsResponseList{}
		err := json.Unmarshal(rawResponse.bodyBytes, response)
		if err != nil {
			return nil, err
		}

		lastUpdated := time.Now()
		for _, result := range response.Results {
			tags = append(tags, &registries.Tag{
				Tag:          result.Name,
				TagTimestamp: result.LastUpdated,
			})
			lastUpdated = result.LastUpdated
		}
		if response.Next == "" || stopAt.After(lastUpdated) {
			continuing = false
		}
		page++
	}

	return tags, nil
}
