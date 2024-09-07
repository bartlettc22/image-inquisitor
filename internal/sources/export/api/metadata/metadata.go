package exportapimetadata

import "time"

type ExportMetadata struct {
	Version    string    `yaml:"version"`
	Kind       string    `yaml:"kind"`
	Created    time.Time `yaml:"created"`
	ExternalID string    `yaml:"externalID"`
}
