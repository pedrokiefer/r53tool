package cli

import (
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/stretchr/testify/require"
)

func TestResourceRecordsToString(t *testing.T) {
	rr := []rtypes.ResourceRecord{{Value: aws.String("a")}, {Value: aws.String("b")}}
	s := resourceRecordsToString(rr)
	require.Equal(t, "a, b", s)
}

func TestMatchInResourceRecord(t *testing.T) {
	re := regexp.MustCompile("foo")

	// Match by Name
	rr := rtypes.ResourceRecordSet{Name: aws.String("foo.example.com.")}
	require.True(t, matchInResourceRecord(re, rr))

	// Match by AliasTarget
	rr = rtypes.ResourceRecordSet{AliasTarget: &rtypes.AliasTarget{DNSName: aws.String("bar.foo.example.com.")}}
	require.True(t, matchInResourceRecord(re, rr))

	// Match by ResourceRecords values
	rr = rtypes.ResourceRecordSet{ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("1.2.3.4")}, {Value: aws.String("foo-target")}}}
	require.True(t, matchInResourceRecord(re, rr))

	// No match
	rr = rtypes.ResourceRecordSet{Name: aws.String("nope.example.com.")}
	require.False(t, matchInResourceRecord(re, rr))
}

func TestNSRecordsToString(t *testing.T) {
	rs := rtypes.ResourceRecordSet{
		ResourceRecords: []rtypes.ResourceRecord{
			{Value: aws.String("ns1.example.net.")},
			{Value: aws.String("ns2.example.net.")},
		},
	}
	s := nsRecordsToString(rs)
	require.Equal(t, "ns1.example.net.,ns2.example.net.", s)
}

func TestHasParkedTag(t *testing.T) {
	tags := []dns.Tag{{Name: "parked", Value: "true"}}
	require.True(t, hasParkedTag(tags))
	tags = []dns.Tag{{Name: "parked", Value: "false"}}
	require.False(t, hasParkedTag(tags))
	tags = []dns.Tag{{Name: "other", Value: "true"}}
	require.False(t, hasParkedTag(tags))
}
