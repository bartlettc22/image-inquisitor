package registries

import (
	"encoding/json"
	"time"
)

type Tag struct {
	Tag          string
	TagTimestamp time.Time
}

// Upgrade states. Defined here because this package owns tag semantics; the
// vulnerability report copies the result rather than reimplementing the
// comparison, so the two cannot drift apart on either spelling or rules.
const (
	UpgradeStateCurrent = "current"
	UpgradeStateCheck   = "check"
	UpgradeStateUnknown = "unknown"
)

// CompareTags reports how a running tag relates to the newest tag a registry
// knows about.
//
// It returns "check" rather than "behind" for a mismatch on purpose. LatestTag
// is the highest semver tag in the repository, and against images known to be
// current it is wrong more often than right: it flags pinned tracks
// (node:24-alpine -> 26.8.2), equivalent tags (busybox:1.38 -> 1.38.0) and
// unrelated tag series (app:v0.17.3 -> v1.6.5). Equality is
// trustworthy; inequality has many benign causes. So "current" is a real signal
// and everything else only means "worth a look".
//
// "unknown" is kept distinct from "check": a registry that could not determine a
// latest tag has told us nothing, which is not the same as being out of date.
func CompareTags(runningTag, latestTag string) string {
	switch {
	case latestTag == "":
		return UpgradeStateUnknown
	case runningTag == latestTag:
		return UpgradeStateCurrent
	default:
		return UpgradeStateCheck
	}
}

// usableTime rejects timestamps that carry no information. Go's zero time (year
// 1) means the field was never set. The Unix epoch means a reproducible build
// pinned it deliberately -- registry.k8s.io images commonly do this. Both
// previously serialised as real dates and rendered in Grafana as
// "0000-12-31 17:00:04", which looks like a corrupt record rather than a missing
// value.
func usableTime(t time.Time) bool {
	return !t.IsZero() && t.Year() > 2000
}

type RegistryImageReport struct {
	Registry           string    `json:"registry"`
	Owner              string    `json:"owner"`
	Repository         string    `json:"repository"`
	Tag                string    `json:"tag"`
	TagTimestamp       time.Time `json:"tagTimestamp"`
	LatestTag          string    `json:"latestSemverTag"`
	LatestTagTimestamp time.Time `json:"latestSemverTagTimestamp"`
}

// UpgradeState is the report's own view of CompareTags.
func (r *RegistryImageReport) UpgradeState() string {
	return CompareTags(r.Tag, r.LatestTag)
}

// MarshalJSON omits timestamps that carry no information and always emits
// upgradeState.
//
// The state is never omitted even when unknown: Loki drops records whose label
// does not exist from a label filter, so an absent field would silently remove
// exactly the images nothing is known about from any filtered view.
func (r RegistryImageReport) MarshalJSON() ([]byte, error) {
	type payload struct {
		Registry           string     `json:"registry"`
		Owner              string     `json:"owner"`
		Repository         string     `json:"repository"`
		Tag                string     `json:"tag"`
		TagTimestamp       *time.Time `json:"tagTimestamp,omitempty"`
		LatestTag          string     `json:"latestSemverTag,omitempty"`
		LatestTagTimestamp *time.Time `json:"latestSemverTagTimestamp,omitempty"`
		UpgradeState       string     `json:"upgradeState"`
	}
	p := payload{
		Registry:     r.Registry,
		Owner:        r.Owner,
		Repository:   r.Repository,
		Tag:          r.Tag,
		LatestTag:    r.LatestTag,
		UpgradeState: CompareTags(r.Tag, r.LatestTag),
	}
	if usableTime(r.TagTimestamp) {
		t := r.TagTimestamp
		p.TagTimestamp = &t
	}
	if usableTime(r.LatestTagTimestamp) {
		t := r.LatestTagTimestamp
		p.LatestTagTimestamp = &t
	}
	return json.Marshal(p)
}
