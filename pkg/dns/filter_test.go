package dns

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/stretchr/testify/require"
)

func TestRemoveResourceRecordsWithTypes(t *testing.T) {
	records := []rtypes.ResourceRecordSet{
		{Name: aws.String("example.com."), Type: rtypes.RRTypeA},
		{Name: aws.String("example.com."), Type: rtypes.RRTypeAaaa},
		{Name: aws.String("www.example.com."), Type: rtypes.RRTypeCname},
		{Name: aws.String("example.com."), Type: rtypes.RRTypeTxt},
	}
	out := RemoveResourceRecordsWithTypes(records, []rtypes.RRType{rtypes.RRTypeA, rtypes.RRTypeAaaa})
	require.Len(t, out, 2)
	require.Equal(t, rtypes.RRTypeCname, out[0].Type)
	require.Equal(t, rtypes.RRTypeTxt, out[1].Type)
}

func TestFindParkedResourceRecord(t *testing.T) {
	records := []rtypes.ResourceRecordSet{
		{Name: aws.String("example.com."), Type: rtypes.RRTypeA},
		{Name: aws.String("example.com."), Type: rtypes.RRTypeAaaa},
		{Name: aws.String("www.example.com."), Type: rtypes.RRTypeCname},
		{Name: aws.String("api.example.com."), Type: rtypes.RRTypeA},
	}

	rr, p := FindParkedResourceRecord(records, "example.com.")

	require.True(t, p.HasARecord)
	require.True(t, p.HasAAAARecord)
	require.True(t, p.HasWWWCnameRecord)

	// rr should contain the three parked records
	require.Len(t, rr, 3)
}

func TestFindNSRecord(t *testing.T) {
	_, err := FindNSRecord([]rtypes.ResourceRecordSet{})
	require.Error(t, err)

	records := []rtypes.ResourceRecordSet{
		{Name: aws.String("example.com."), Type: rtypes.RRTypeTxt},
		{Name: aws.String("example.com."), Type: rtypes.RRTypeNs},
	}
	ns, err := FindNSRecord(records)
	require.NoError(t, err)
	require.Equal(t, rtypes.RRTypeNs, ns.Type)
}
