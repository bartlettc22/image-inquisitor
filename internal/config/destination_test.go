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
			destination: "file://tmp/path/to/file.yaml",
			protocol:    "file",
			path:        "tmp/path/to/file.yaml",
			errContains: "",
		},
		{
			destination: "file://tmp/path/to/dir/",
			protocol:    "file",
			path:        "tmp/path/to/dir/",
			errContains: "",
		},
		{
			destination: "file:///tmp/path/from/root/dir/file.yaml",
			protocol:    "file",
			path:        "/tmp/path/from/root/dir/file.yaml",
			errContains: "",
		},
		{
			destination: "gs://bucket/path/to/file.yaml",
			protocol:    "gs",
			path:        "bucket/path/to/file.yaml",
			errContains: "",
		},
		{
			destination: "gs://bucket/path/to/dir/",
			protocol:    "gs",
			path:        "bucket/path/to/dir/",
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
		protocol, path, err := ParseDestination(test.destination)
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
