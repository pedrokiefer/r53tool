package vuln

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDMARCScan_NoIssues(t *testing.T) {
	issues := dmarcScan("v=DMARC1; p=reject; sp=quarantine; pct=100")
	require.Empty(t, issues)
}

func TestDMARCScan_InvalidPct_NonInt(t *testing.T) {
	issues := dmarcScan("v=DMARC1; pct=abc")
	require.NotEmpty(t, issues)
	found := false
	for _, is := range issues {
		if contains(is, []string{"pct has invalid value"}) {
			found = true
		}
	}
	require.True(t, found)
}

func TestDMARCScan_PctOver100(t *testing.T) {
	issues := dmarcScan("v=DMARC1; pct=150")
	require.NotEmpty(t, issues)
	found := false
	for _, is := range issues {
		if contains(is, []string{"pct has invalid value"}) {
			found = true
		}
	}
	require.True(t, found)
}
