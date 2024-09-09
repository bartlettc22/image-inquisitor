package exportapimetadata

import "time"

const (
	Kind = "ExportReport"
)

type ExportMetadata struct {
	Version  string    `yaml:"version"`
	Kind     string    `yaml:"kind"`
	Created  time.Time `yaml:"created"`
	SourceID string    `yaml:"sourceID"`
}
