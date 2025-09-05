package dns

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	rdtypes "github.com/aws/aws-sdk-go-v2/service/route53domains/types"
	"github.com/stretchr/testify/require"
)

func TestMatchNSRecords(t *testing.T) {
	rs := rtypes.ResourceRecordSet{
		Type: rtypes.RRTypeNs,
		ResourceRecords: []rtypes.ResourceRecord{
			{Value: aws.String("ns1.example.net.")},
			{Value: aws.String("ns2.example.net.")},
		},
	}
	ns := []string{"ns1.example.net", "ns2.example.net"}
	require.True(t, MatchNSRecords(ns, rs))

	nsMissing := []string{"ns1.example.net", "ns3.example.net"}
	require.False(t, MatchNSRecords(nsMissing, rs))
}

func TestNameserversFromRecords(t *testing.T) {
	rs := rtypes.ResourceRecordSet{
		Type: rtypes.RRTypeNs,
		ResourceRecords: []rtypes.ResourceRecord{
			{Value: aws.String("ns1.example.net.")},
			{Value: aws.String("ns2.example.net.")},
		},
	}
	got := nameserversFromRecords(rs)
	require.Equal(t, []rdtypes.Nameserver{
		{Name: aws.String("ns1.example.net")},
		{Name: aws.String("ns2.example.net")},
	}, got)
}

func TestFindInList(t *testing.T) {
	list := []string{"a", "b", "c"}
	require.True(t, findInList(list, "b"))
	require.False(t, findInList(list, "d"))
}
