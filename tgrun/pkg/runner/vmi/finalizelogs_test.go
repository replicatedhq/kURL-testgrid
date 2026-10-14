package vmi

import (
	"regexp"
	"strconv"
	"testing"
)

// TestFinalizeLogsRetryTimeoutExceedsCurlBudget guards against sc-140012
// regressing: the retry's outer `timeout` must exceed curl's own worst-case
// duration (--connect-timeout + --max-time), otherwise the outer timeout
// fires first and kills the very upload the retry exists to rescue.
func TestFinalizeLogsRetryTimeoutExceedsCurlBudget(t *testing.T) {
	script := string(finalizeLogs)

	connectTimeout := mustExtractInt(t, script, `--connect-timeout\s+(\d+)`)
	maxTime := mustExtractInt(t, script, `--max-time\s+(\d+)`)

	retryTimeoutRe := regexp.MustCompile(`timeout\s+(\d+)\s+bash[^\n]*send_logs"\s*\|\|\s*true`)
	match := retryTimeoutRe.FindStringSubmatch(script)
	if match == nil {
		t.Fatalf("could not find retry timeout statement in finalizelogs.sh")
	}
	retryTimeout, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("could not parse retry timeout %q: %v", match[1], err)
	}

	curlBudget := connectTimeout + maxTime
	if retryTimeout <= curlBudget {
		t.Fatalf("retry timeout %ds does not exceed curl's worst-case budget %ds (connect-timeout %ds + max-time %ds); the outer timeout will always fire first",
			retryTimeout, curlBudget, connectTimeout, maxTime)
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
