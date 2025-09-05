package dns

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/require"
)

func TestCreateChanges_ExcludesApexNSAndSOA_IncludesOthers(t *testing.T) {
	domain := "example.com"
	apex := NormalizeDomain(domain)

	input := []rtypes.ResourceRecordSet{
		{ // apex NS should be skipped
			Name:            aws.String(apex),
			Type:            rtypes.RRTypeNs,
			ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("ns1.example.net.")}},
		},
		{ // apex SOA should be skipped
			Name:            aws.String(apex),
			Type:            rtypes.RRTypeSoa,
			ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("ns1.example.net. hostmaster.example.com. 1 7200 3600 1209600 3600")}},
		},
		{ // subdomain NS should be included
			Name:            aws.String("sub." + apex),
			Type:            rtypes.RRTypeNs,
			ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("ns2.example.net.")}},
			TTL:             aws.Int64(60),
		},
		{ // TXT should be included
			Name:            aws.String(apex),
			Type:            rtypes.RRTypeTxt,
			ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("\"hello\"")}},
			TTL:             aws.Int64(300),
		},
	}

	changes := NewRouteManager(context.TODO(), "", &RouteManagerOptions{}).CreateChanges(domain, input)

	// Expect 2 changes: sub NS and TXT
	require.Len(t, changes, 2)
	types := []rtypes.RRType{changes[0].ResourceRecordSet.Type, changes[1].ResourceRecordSet.Type}
	require.Contains(t, types, rtypes.RRTypeNs)
	require.Contains(t, types, rtypes.RRTypeTxt)
}
