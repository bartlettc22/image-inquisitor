package registries

import (
	"time"
)

type Tag struct {
	Tag          string
	TagTimestamp time.Time
}

type RegistryImageReport struct {
	Registry           string    `json:"registry"`
	Owner              string    `json:"owner"`
	Repository         string    `json:"repository"`
	Tag                string    `json:"tag"`
	TagTimestamp       time.Time `json:"tagTimestamp"`
	LatestTag          string    `json:"latestSemverTag"`
	LatestTagTimestamp time.Time `json:"latestSemverTagTimestamp"`
}
