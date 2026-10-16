package vmi

import (
	"regexp"
	"strconv"
	"testing"
)

// TestFinalizeLogsRetryTimeoutExceedsCurlBudget guards against sc-140012
// regressing: every outer `timeout` wrapping send_logs -- the first attempt
// as well as the retry -- must exceed curl's own worst-case duration
// (--connect-timeout + --max-time), otherwise the outer timeout fires first
// and kills the very upload it wraps.
func TestFinalizeLogsRetryTimeoutExceedsCurlBudget(t *testing.T) {
	script := string(finalizeLogs)

	connectTimeout := mustExtractInt(t, script, `--connect-timeout\s+(\d+)`)
	maxTime := mustExtractInt(t, script, `--max-time\s+(\d+)`)
	curlBudget := connectTimeout + maxTime

	timeoutRe := regexp.MustCompile(`timeout\s+(\d+)\s+bash[^\n]*send_logs"`)
	matches := timeoutRe.FindAllStringSubmatch(script, -1)
	if len(matches) < 2 {
		t.Fatalf("expected to find both the first attempt's and the retry's timeout statements in finalizelogs.sh, found %d", len(matches))
	}

	for _, match := range matches {
		timeoutSeconds, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatalf("could not parse timeout %q: %v", match[1], err)
		}
		if timeoutSeconds <= curlBudget {
			t.Fatalf("timeout %ds does not exceed curl's worst-case budget %ds (connect-timeout %ds + max-time %ds); the outer timeout will always fire first",
				timeoutSeconds, curlBudget, connectTimeout, maxTime)
		}
	}
}

func mustExtractInt(t *testing.T, script, pattern string) int {
	t.Helper()
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(script)
	if match == nil {
		t.Fatalf("pattern %q not found in finalizelogs.sh", pattern)
	}
	n, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("could not parse %q as int: %v", match[1], err)
	}
	return n
}
