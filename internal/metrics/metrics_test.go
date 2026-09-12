package metrics

import (
	"strings"
	"testing"
)

func TestEscapesLabelValues(t *testing.T) {
	p := NewPublisher("")
	// Image refs come from cluster specs; an unescaped quote or backslash would
	// produce a malformed payload and the whole push would be rejected, silently
	// losing an entire scan.
	p.Gauge("m", "h", map[string]string{"image": `we"ird\path`}, 1)
	got := p.Payload()
	if !strings.Contains(got, `image="we\"ird\\path"`) {
		t.Errorf("label value not escaped:\n%s", got)
	}
}

func TestLabelsSortedAndHelpEmittedOnce(t *testing.T) {
	p := NewPublisher("")
	p.Gauge("m", "h", map[string]string{"z": "1", "a": "2"}, 1)
	p.Gauge("m", "h", map[string]string{"z": "3", "a": "4"}, 2)
	got := p.Payload()
	if strings.Count(got, "# HELP m") != 1 {
		t.Errorf("HELP should appear once:\n%s", got)
	}
	if !strings.Contains(got, `m{a="2",z="1"} 1`) {
		t.Errorf("labels should be sorted:\n%s", got)
	}
}

func TestNoLabelsFormsBareSample(t *testing.T) {
	p := NewPublisher("")
	p.Gauge("scan_ts", "h", nil, 1789187833)
	if !strings.Contains(p.Payload(), "scan_ts 1.789187833e+09") {
		t.Errorf("bare sample malformed:\n%s", p.Payload())
	}
}

func TestEmptyPayloadIsRefused(t *testing.T) {
	// Pushing nothing would look like a successful scan that found no images.
	p := NewPublisher("http://example.invalid")
	if err := p.Push(0); err == nil {
		t.Error("an empty payload must be refused, not pushed")
	}
}

func TestNoEndpointIsNotAnError(t *testing.T) {
	p := NewPublisher("")
	p.Gauge("m", "h", nil, 1)
	if err := p.Push(0); err != nil {
		t.Errorf("publishing should be a no-op when disabled, got %v", err)
	}
}
