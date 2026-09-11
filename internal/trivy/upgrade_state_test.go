package trivy

import "testing"

func TestSetUpgradeContext(t *testing.T) {
	cases := []struct{ running, latest, want string }{
		{"v2.15.2", "v2.15.2", UpgradeStateCurrent}, // harbor after the upgrade
		{"v1.11.1", "v1.12.1", UpgradeStateCheck},   // longhorn, genuinely behind
		{"1.38", "1.38.0", UpgradeStateCheck},       // false positive, still only "check"
		{"latest-full", "", UpgradeStateUnknown},    // registry could not determine
		{"", "", UpgradeStateUnknown},
	}
	for _, c := range cases {
		r := &TrivyImageReport{}
		r.SetUpgradeContext(c.running, c.latest)
		if r.UpgradeState != c.want {
			t.Errorf("running=%q latest=%q got %q want %q", c.running, c.latest, r.UpgradeState, c.want)
		}
		if r.RunningTag != c.running || r.LatestTag != c.latest {
			t.Errorf("tags not recorded for %q/%q", c.running, c.latest)
		}
	}
}
