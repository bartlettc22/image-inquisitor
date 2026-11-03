package versions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatest(t *testing.T) {
	tags := []string{
		"latest", "16", "17", "16-alpine", "17-alpine",
		"16.2", "16.4", "17.1", "17.2-alpine", "17.2-bookworm", "18rc1", "18.0-rc1",
		"16.4.1", "16.4.2", "17.0.0", "608111629",
		"v1.2.0", "v1.10.0", "v1.11.0-distroless", "v1.12.0-distroless",
		"16.2.0-debian-12-r6", "16.2.0-debian-12-r30", "16.3.0-debian-12-r2",
		"v2.15.0-20251030", "v2.15.0-20251201", "v2.16.0-20251101",
	}

	cases := []struct {
		current, latest string
		ok              bool
	}{
		{"16", "17", true},
		{"16-alpine", "17-alpine", true},
		{"16.2", "17.1", true},
		{"17.1-alpine", "17.2-alpine", true},
		{"16.4.1", "17.0.0", true},
		{"17.0.0", "17.0.0", true},
		{"v1.2.0", "v1.10.0", true},
		{"v1.11.0-distroless", "v1.12.0-distroless", true},
		{"16.2.0-debian-12-r6", "16.3.0-debian-12-r2", true},
		{"v2.15.0-20251030", "v2.16.0-20251101", true},
		{"9", "17", true},
		{"20.0.0", "20.0.0", true},
		{"latest", "", false},
		{"stable-alpine", "", false},
		{"sha-24af8f01", "", false},
	}
	for _, c := range cases {
		latest, ok := Latest(c.current, tags)
		assert.Equal(t, c.ok, ok, c.current)
		assert.Equal(t, c.latest, latest, c.current)
	}
}

func TestLatestCalendarVersions(t *testing.T) {
	// Some images have old calendar-versioned tags alongside semantic versions
	tags := []string{"1.34.0", "1.35.1", "2021.12.12", "2025.8.1", "2026.10.1"}

	latest, _ := Latest("1.34.0", tags)
	assert.Equal(t, "1.35.1", latest)
	latest, _ = Latest("2025.8.1", tags)
	assert.Equal(t, "2026.10.1", latest)
}

func TestVersions(t *testing.T) {
	assert.Equal(t, []string{"1.2", "v3-alpine"}, Versions([]string{"latest", "1.2", "pr-1234", "v3-alpine", "sha-abc"}))
}

func TestBump(t *testing.T) {
	assert.Equal(t, BumpMajor, Bump("16", "17"))
	assert.Equal(t, BumpMinor, Bump("v1.2.0", "v1.10.0"))
	assert.Equal(t, BumpPatch, Bump("1.2.3-alpine", "1.2.4-alpine"))
	assert.Equal(t, BumpOther, Bump("16.2.0-debian-12-r6", "16.2.0-debian-12-r30"))
	assert.Equal(t, "", Bump("1.2.3", "1.2.3"))
	assert.Equal(t, "", Bump("1.2", "1.2.3"), "different precision")
	assert.Equal(t, "", Bump("latest", "1.2.3"))
}
