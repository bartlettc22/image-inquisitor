// Package versions finds the newest version of an image tag or chart version among a
// repository's tags, keeping to the same variant: "16.2-alpine" is only compared with
// other "<major>.<minor>-alpine" tags, as Renovate does
package versions

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	BumpMajor = "major"
	BumpMinor = "minor"
	BumpPatch = "patch"

	// BumpOther is a newer tag that differs only after the version numbers,
	// e.g. a rebuild ("-r6" to "-r7") or a dated suffix
	BumpOther = "other"
)

var (
	tagRe   = regexp.MustCompile(`^(v?)(\d+(?:\.\d+)*)(.*)$`)
	digitRe = regexp.MustCompile(`\d+`)
)

// version is a parsed tag: "v1.2.3-debian-12-r6" has prefix "v", numbers [1 2 3]
// and suffix "-debian-12-r6"
type version struct {
	numbers []int
	suffix  []int
	variant string
}

func parse(tag string) (version, bool) {
	m := tagRe.FindStringSubmatch(tag)
	if m == nil {
		return version{}, false
	}
	parts := strings.Split(m[2], ".")
	v := version{numbers: make([]int, len(parts))}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return version{}, false
		}
		v.numbers[i] = n
	}
	for _, d := range digitRe.FindAllString(m[3], -1) {
		n, err := strconv.Atoi(d)
		if err != nil {
			return version{}, false
		}
		v.suffix = append(v.suffix, n)
	}

	// Same prefix, precision, leading number width and suffix with its numbers masked
	v.variant = strings.Join([]string{m[1], strconv.Itoa(len(parts)), width(parts[0]), digitRe.ReplaceAllString(m[3], "#")}, "|")
	return v, true
}

// width classes a leading number, so a semantic version never compares with a calendar
// version ("2021.12.12", as old tags of some images are) or a date or build ID ("20251030")
func width(n string) string {
	switch {
	case len(n) <= 3:
		return "short"
	case len(n) <= 5:
		return "year"
	default:
		return "long"
	}
}

func (v version) compare(o version) int {
	if c := slices.Compare(v.numbers, o.numbers); c != 0 {
		return c
	}
	return slices.Compare(v.suffix, o.suffix)
}

// Versions returns the tags that are versions, dropping ones Latest never picks
// ("latest", commit SHAs, "pr-1234"), which can be most of a busy repository's tags
func Versions(tags []string) []string {
	kept := make([]string, 0, len(tags))
	for _, t := range tags {
		if _, ok := parse(t); ok {
			kept = append(kept, t)
		}
	}
	return kept
}

// Latest returns the newest tag of the same variant as current, which is current itself when
// it is the newest. ok is false when current isn't a version (e.g. "latest", a commit SHA).
func Latest(current string, tags []string) (latest string, ok bool) {
	cur, ok := parse(current)
	if !ok {
		return "", false
	}

	latest = current
	best := cur
	for _, tag := range tags {
		v, ok := parse(tag)
		if !ok || v.variant != cur.variant {
			continue
		}
		if v.compare(best) > 0 {
			latest, best = tag, v
		}
	}
	return latest, true
}

// Bump classifies the difference between two versions of the same variant:
// major, minor, patch or other. It returns "" if they aren't comparable or equal.
func Bump(from, to string) string {
	f, okF := parse(from)
	t, okT := parse(to)
	if !okF || !okT || f.variant != t.variant || f.compare(t) == 0 {
		return ""
	}
	for i := range f.numbers {
		if f.numbers[i] != t.numbers[i] {
			switch i {
			case 0:
				return BumpMajor
			case 1:
				return BumpMinor
			default:
				return BumpPatch
			}
		}
	}
	return BumpOther
}
