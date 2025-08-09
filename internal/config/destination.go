package config

import (
	"fmt"
	"strings"
)

// Destination is a string representation of a destination for importing or exporting
type Destination string

const (

	// DestinationFile is a file/directory destination
	DestinationFile Destination = "file"

	// DestinationGCS is a GCS bucket destination
	DestinationGCS Destination = "gs"
)

func ParseDestination(destination string) (Destination, string, error) {

	parts := strings.Split(destination, "://")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid destination: %s", destination)
	}

	switch protocol := parts[0]; protocol {
	case DestinationFile.String():
		return DestinationFile, parts[1], nil
	case DestinationGCS.String():
		return DestinationGCS, parts[1], nil
	default:
		return "", "", fmt.Errorf("invalid protocol '%s' in destination: %s", protocol, destination)
	}
}

func (d Destination) String() string {
	return string(d)
}
