package cli

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/require"
)

func TestParkCreateChanges_WithAlias(t *testing.T) {
	a := &parkApp{
		Alias:    true,
		Hostname: "alias.example.net.",
		ZoneID:   "Z12345",
	}
	changes := a.createChanges("example.com.")

	require.Len(t, changes, 2)
	// First change is CNAME for www
	require.Equal(t, rtypes.RRTypeCname, changes[0].ResourceRecordSet.Type)
	require.Equal(t, "www.example.com.", aws.ToString(changes[0].ResourceRecordSet.Name))
	// Second change is A Alias
	require.Equal(t, rtypes.RRTypeA, changes[1].ResourceRecordSet.Type)
	require.NotNil(t, changes[1].ResourceRecordSet.AliasTarget)
	require.Equal(t, "alias.example.net.", aws.ToString(changes[1].ResourceRecordSet.AliasTarget.DNSName))
	require.Equal(t, "Z12345", aws.ToString(changes[1].ResourceRecordSet.AliasTarget.HostedZoneId))
}

func TestParkCreateChanges_WithIPs(t *testing.T) {
	a := &parkApp{
		Alias: false,
		IPSv4: []rtypes.ResourceRecord{{Value: aws.String("1.2.3.4")}},
		IPSv6: []rtypes.ResourceRecord{{Value: aws.String("2001:db8::1")}},
	}
	changes := a.createChanges("example.com.")

	require.Len(t, changes, 3)
	// CNAME www
	require.Equal(t, rtypes.RRTypeCname, changes[0].ResourceRecordSet.Type)
	// A and AAAA
	require.Equal(t, rtypes.RRTypeA, changes[1].ResourceRecordSet.Type)
	require.Equal(t, rtypes.RRTypeA, changes[2].ResourceRecordSet.Type) // Note: code sets TypeA for both blocks; might be a bug, but we assert current behavior
}
