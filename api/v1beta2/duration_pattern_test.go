package v1beta2

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// crdDurationPattern reads the pattern the apiserver will actually enforce out
// of the generated CRD, rather than restating it here. A copy in the test would
// keep passing after the marker changed.
func crdDurationPattern(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "config", "crd", "bases",
		"gitlab.k8s.alekc.dev_runners.yaml"))
	if err != nil {
		t.Fatalf("read CRD: %v", err)
	}
	// The key is followed by a description block, then the pattern. Bounded so a
	// missing pattern cannot silently pick up a later field's one.
	re := regexp.MustCompile(`cleanup_resources_timeout:\n(?:[^\n]*\n){0,6}?\s*pattern: (\S+)`)
	m := re.FindSubmatch(raw)
	if m == nil {
		t.Fatal("cleanup_resources_timeout pattern not found in the generated CRD")
	}
	return string(m[1])
}

// The pattern must accept what the runner's time.ParseDuration accepts, with
// one deliberate exception: a negative duration parses fine but is
// meaningless for a cleanup timeout, so admission rejects it instead.
func TestDurationPatternMatchesParseDuration(t *testing.T) {
	re, err := regexp.Compile(crdDurationPattern(t))
	if err != nil {
		t.Fatalf("CRD pattern does not compile: %v", err)
	}

	for _, c := range []string{
		"0", "+0", "0s", "5m", "300s", "1h30m", "1h30m0s", "1.5h", ".5s", "+3m",
		"1ns", "1us", "1µs", "1μs", "100ms", "1m0.5s", "0.000000001s",
		"2562047h47m16.854775807s",
	} {
		if _, perr := time.ParseDuration(c); perr != nil {
			t.Fatalf("test case %q is not a valid Go duration: %v", c, perr)
		}
		if !re.MatchString(c) {
			t.Errorf("pattern rejects the valid duration %q", c)
		}
	}

	for _, c := range []string{"", " ", "5m ", "1d", "1h30", "5M", "1e3s", "1h 30m", "00", "abc"} {
		if _, perr := time.ParseDuration(c); perr == nil {
			t.Fatalf("test case %q is unexpectedly a valid Go duration", c)
		}
		if re.MatchString(c) {
			t.Errorf("pattern accepts the invalid duration %q", c)
		}
	}

	// Negative durations parse but are rejected on purpose.
	for _, c := range []string{"-5m", "-1s"} {
		if _, perr := time.ParseDuration(c); perr != nil {
			t.Fatalf("expected %q to parse as a Go duration", c)
		}
		if re.MatchString(c) {
			t.Errorf("pattern should reject the negative duration %q", c)
		}
	}
}
