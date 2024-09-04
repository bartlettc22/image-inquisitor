package registries

import (
	"time"
)

type Tag struct {
	Tag          string
	TagTimestamp time.Time
}

type RegistryImageReport struct {
	CurrentTag          string
	CurrentTagTimestamp time.Time
	LatestTag           string
	LatestTagTimestamp  time.Time
}
