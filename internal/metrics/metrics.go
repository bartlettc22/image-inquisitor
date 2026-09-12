// Package metrics publishes scan results as Prometheus time series.
//
// Why this exists alongside the log reports: logs have no concept of "current
// state". Every consumer has to ask "which scan falls inside this time window",
// and that question produced a long tail of wrong answers -- a dashboard window
// shorter than the scan interval showed nothing for most of the day, a window
// exactly equal to it went blank while a scan was running, and a wider window
// needed deduplication to avoid showing each image once per scan. Metrics have
// last-value semantics, so none of that arises.
//
// Logs remain the right place for the raw scan record and for per-CVE detail.
// They are good at "what happened" and bad at "what is true now".
package metrics

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Publisher accumulates samples and writes them to a Prometheus-compatible
// import endpoint.
type Publisher struct {
	endpoint string
	buf      bytes.Buffer
	seen     map[string]bool
}

func NewPublisher(endpoint string) *Publisher {
	return &Publisher{endpoint: endpoint, seen: map[string]bool{}}
}

// escape applies Prometheus exposition-format escaping to a label VALUE.
// Image references are user-controlled strings that reach us from cluster specs,
// so an unescaped quote or backslash would produce a malformed payload and the
// whole push would be rejected -- silently losing an entire scan.
func escape(v string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return r.Replace(v)
}

// help writes HELP/TYPE once per metric name.
func (p *Publisher) help(name, typ, text string) {
	if p.seen[name] {
		return
	}
	p.seen[name] = true
	fmt.Fprintf(&p.buf, "# HELP %s %s\n# TYPE %s %s\n", name, text, name, typ)
}

// Gauge records one sample. Labels are emitted in sorted order so identical
// series produce byte-identical output, which makes the payload diffable.
func (p *Publisher) Gauge(name, helpText string, labels map[string]string, value float64) {
	p.help(name, "gauge", helpText)
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		// %q would escape a second time on top of escape(), turning we"ird into
		// we\\"ird. Quote explicitly so escaping happens exactly once.
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, escape(labels[k])))
	}
	if len(parts) == 0 {
		fmt.Fprintf(&p.buf, "%s %g\n", name, value)
		return
	}
	fmt.Fprintf(&p.buf, "%s{%s} %g\n", name, strings.Join(parts, ","), value)
}

// Payload returns what would be pushed. Exposed for tests and for --dry-run.
func (p *Publisher) Payload() string { return p.buf.String() }

// Push sends the accumulated samples.
//
// A failure here must never fail the scan: the log reports are the system of
// record and are already written by this point. The caller logs the error and
// carries on.
func (p *Publisher) Push(timeout time.Duration) error {
	if p.endpoint == "" {
		return nil
	}
	if p.buf.Len() == 0 {
		return fmt.Errorf("refusing to push an empty payload")
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Post(p.endpoint, "text/plain", bytes.NewReader(p.buf.Bytes()))
	if err != nil {
		return fmt.Errorf("pushing metrics to %s: %w", p.endpoint, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("pushing metrics to %s: HTTP %d: %s", p.endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
