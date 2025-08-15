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
		},
		{
			destination: "file://tmp/path/to/dir/",
			protocol:    "file",
			path:        "tmp/path/to/dir/",
		},
		{
			destination: "file:///tmp/path/from/root/dir/file.yaml",
			protocol:    "file",
			path:        "/tmp/path/from/root/dir/file.yaml",
		},
		{
			destination: "gs://bucket/path/to/file.yaml",
			protocol:    "gs",
			path:        "bucket/path/to/file.yaml",
		},
		{
			destination: "gs://bucket/path/to/dir/",
			protocol:    "gs",
			path:        "bucket/path/to/dir/",
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
			errContains: "invalid protocol",
		},
		{
			destination: "file://lib://bucket/path/to/file.yaml",
			protocol:    "",
			path:        "",
			errContains: "invalid file destination",
		},
		{
			destination: "stdout",
			protocol:    "stdout",
			path:        "",
		},
		{
			destination: "kubernetes",
			protocol:    "kubernetes",
			path:        "",
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
