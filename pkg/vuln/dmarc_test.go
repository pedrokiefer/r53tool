package vuln

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/require"
)

func TestCheckDMARC_NoRecord(t *testing.T) {
	issues := CheckDMARC("example.com", nil)
	require.NotEmpty(t, issues)
	require.Contains(t, issues[0], "is a MX with no DMARC record")
}

func TestCheckDMARC_PolicyWarnings(t *testing.T) {
	rs := []rtypes.ResourceRecordSet{
		{
			Name: aws.String("_dmarc.example.com"),
			Type: rtypes.RRTypeTxt,
			ResourceRecords: []rtypes.ResourceRecord{
				{Value: aws.String("v=DMARC1; p=none; sp=none; pct=50")},
			},
		},
	}
	issues := CheckDMARC("example.com", rs)
	// Expect 3 warnings: p, sp, and pct<100
	foundP := false
	foundSP := false
	foundPct := false
	for _, is := range issues {
		if contains(is, []string{"DMARC policy is", "allows spoofed emails"}) {
			foundP = true
		}
		if contains(is, []string{"DMARC subdomain policy is", "allows spoofed emails"}) {
			foundSP = true
		}
		if contains(is, []string{"DMARC policy is only applied to", "%"}) {
			foundPct = true
		}
	}
	require.True(t, foundP && foundSP && foundPct)
}

func contains(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
