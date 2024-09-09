package types

import (
	"fmt"
	"strings"
)

type ExportableReport interface {
	Export() map[string]interface{}
}

type ExportDestination string

const (
	ExportDestinationFile ExportDestination = "file"
	ExportDestinationGCS  ExportDestination = "gcs"
)

func (e ExportDestination) String() string {
	return string(e)
}

type ExportDestinationList map[string]ExportDestination

func (e ExportDestinationList) Add(dest string) error {
	switch dest {
	case string(ExportDestinationFile):
	case string(ExportDestinationGCS):
	default:
		return fmt.Errorf("invalid export destination: %s", dest)
	}
	e[dest] = ExportDestination(dest)
	return nil
}

func (e ExportDestinationList) String() string {
	dests := []string{}
	for _, dest := range e {
		dests = append(dests, dest.String())
	}
	return strings.Join(dests, ",")
}

func (e ExportDestinationList) Contains(find ExportDestination) bool {
	_, ok := e[string(find)]
	return ok
}
