package registries

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCompareTags(t *testing.T) {
	cases := []struct{ running, latest, want string }{
		{"v2.15.2", "v2.15.2", UpgradeStateCurrent},
		{"v1.11.1", "v1.12.1", UpgradeStateCheck},
		{"1.38", "1.38.0", UpgradeStateCheck},
		{"latest-full", "", UpgradeStateUnknown},
	}
	for _, c := range cases {
		if got := CompareTags(c.running, c.latest); got != c.want {
			t.Errorf("CompareTags(%q,%q)=%q want %q", c.running, c.latest, got, c.want)
		}
	}
}

func TestMarshalOmitsUnusableTimestamps(t *testing.T) {
	epoch := time.Unix(0, 0).UTC()
	r := RegistryImageReport{Tag: "v0.21.0", TagTimestamp: epoch} // zero LatestTagTimestamp
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"tagTimestamp", "latestSemverTagTimestamp"} {
		if _, present := m[k]; present {
			t.Errorf("%s should be omitted, got %v", k, m[k])
		}
	}
	if m["upgradeState"] != UpgradeStateUnknown {
		t.Errorf("upgradeState = %v, want %q (must always be present)", m["upgradeState"], UpgradeStateUnknown)
	}
}

func TestMarshalKeepsRealTimestamps(t *testing.T) {
	real := time.Date(2026, 6, 16, 0, 1, 29, 0, time.UTC)
	r := RegistryImageReport{Tag: "3.24", TagTimestamp: real, LatestTag: "3.24"}
	b, _ := json.Marshal(r)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	if _, present := m["tagTimestamp"]; !present {
		t.Error("a real tagTimestamp must survive")
	}
	if m["upgradeState"] != UpgradeStateCurrent {
		t.Errorf("upgradeState = %v, want current", m["upgradeState"])
	}
}
