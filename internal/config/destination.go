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

	// DestinationStdout is a stdout destination
	DestinationStdout Destination = "stdout"

	// DestinationKubernetes is a Kubernetes destination
	DestinationKubernetes Destination = "kubernetes"
)

func (d Destination) String() string {
	return string(d)
}

func ParseDestination(destination string) (Destination, string, error) {

	parts := strings.Split(destination, "://")

	switch protocol := parts[0]; protocol {
	case DestinationFile.String():
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid file destination: %s", destination)
		}
		return DestinationFile, parts[1], nil
	case DestinationGCS.String():
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GCS destination: %s", destination)
		}
		return DestinationGCS, parts[1], nil
	case DestinationStdout.String():
		return DestinationStdout, "", nil
	case DestinationKubernetes.String():
		return DestinationKubernetes, "", nil
	default:
		return "", "", fmt.Errorf("invalid protocol '%s' in destination: %s", protocol, destination)
	}
}
