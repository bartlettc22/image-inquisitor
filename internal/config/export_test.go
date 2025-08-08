package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDestination(t *testing.T) {

	testmatrix := []struct {
		destination string
		protocol    string
		path        string
		errContains string
	}{
		{
			destination: "file://tmp/image-inquisitor.yaml",
			protocol:    "file",
			path:        "/tmp/image-inquisitor.yaml",
			errContains: "",
		},
		{
			destination: "gs://bucket/path/to/file.yaml",
			protocol:    "gs",
			path:        "bucket/path/to/file.yaml",
			errContains: "",
		},
		{
			destination: "xyz://bucket/path/to/file.yaml",
			protocol:    "",
			path:        "",
			errContains: "invalid protocol",
		},
		{
			destination: "/bucket/path/to/file.yaml",
			protocol:    "",
			path:        "",
			errContains: "invalid destination",
		},
		{
			destination: "file://lib://bucket/path/to/file.yaml",
			protocol:    "",
			path:        "",
			errContains: "invalid destination",
		},
	}

	for _, test := range testmatrix {
		protocol, path, err := ParseExportDestination(test.destination)
		if test.errContains != "" {
			require.Error(t, err)
			require.Contains(t, err.Error(), test.errContains)
			continue
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, test.protocol, protocol.String())
		require.Equal(t, test.path, path)
	}
}
