package registries

import (
	"time"
)

type Tag struct {
	Tag          string
	TagTimestamp time.Time
}

type ImageReport struct {
	CurrentTag          string
	CurrentTagTimestamp time.Time
	CurrentTagAge       time.Duration
	LatestTag           string
	LatestTagTimestamp  time.Time
	LatestTagAge        time.Duration
}