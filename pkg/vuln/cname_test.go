package vuln

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestCheckAliasS3_NoSuchBucket(t *testing.T) {
	httpmock.ActivateNonDefault(cli)
	defer httpmock.DeactivateAndReset()

	// Simulate a 404 NoSuchBucket for alias target
	httpmock.RegisterResponder("GET", "http://bucket.s3.amazonaws.com",
		httpmock.NewStringResponder(404, "Code: NoSuchBucket"))

	f := &Findings{Name: "zone"}
	rs := rtypes.ResourceRecordSet{
		Name:        aws.String("name.example.com"),
		Type:        rtypes.RRTypeA,
		AliasTarget: &rtypes.AliasTarget{DNSName: aws.String("bucket.s3.amazonaws.com")},
	}

	checkAliasS3(context.Background(), f, rs)

	require.Len(t, f.VulnerableRecords, 1)
}

func TestCheckAliasCloudFront_NoSuchBucket(t *testing.T) {
	httpmock.ActivateNonDefault(cli)
	defer httpmock.DeactivateAndReset()

	// checkAliasCloudFront checks the bucket existence using the record name
	httpmock.RegisterResponder("GET", "http://name.example.com",
		httpmock.NewStringResponder(404, "Code: NoSuchBucket"))

	f := &Findings{Name: "zone"}
	rs := rtypes.ResourceRecordSet{
		Name:        aws.String("name.example.com"),
		Type:        rtypes.RRTypeA,
		AliasTarget: &rtypes.AliasTarget{DNSName: aws.String("dxxxx.cloudfront.net")},
	}

	checkAliasCloudFront(context.Background(), f, rs)

	require.Len(t, f.VulnerableRecords, 1)
}
