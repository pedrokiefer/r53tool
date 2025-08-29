package vuln

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/require"
)

func TestCheckSPF_NoTXT(t *testing.T) {
	issues := CheckSPF("example.com", nil)
	require.Len(t, issues, 1)
	require.Contains(t, issues[0], "has no TXT record")
}

func TestCheckSPF_MissingSPF(t *testing.T) {
	rs := []rtypes.ResourceRecordSet{
		{
			Name: aws.String("example.com"),
			Type: rtypes.RRTypeTxt,
			ResourceRecords: []rtypes.ResourceRecord{
				{Value: aws.String("some text without spf")},
			},
		},
	}
	issues := CheckSPF("example.com", rs)
	require.NotEmpty(t, issues)
	require.Contains(t, issues[0], "MX is missing SPF record")
}

func TestCheckSPF_MultipleSPFRecords(t *testing.T) {
	rs := []rtypes.ResourceRecordSet{
		{
			Name: aws.String("example.com"),
			Type: rtypes.RRTypeTxt,
			ResourceRecords: []rtypes.ResourceRecord{
				{Value: aws.String("v=spf1 include:_spf.example.com ~all")},
			},
		},
		{
			Name: aws.String("example.com"),
			Type: rtypes.RRTypeTxt,
			ResourceRecords: []rtypes.ResourceRecord{
				{Value: aws.String("v=spf1 -all")},
			},
		},
	}
	issues := CheckSPF("example.com", rs)
	require.NotEmpty(t, issues)
	found := false
	for _, is := range issues {
		if strings.Contains(is, "has multiple SPF records") {
			found = true
			break
		}
	}
	require.True(t, found, "expected multiple SPF records message")
}
