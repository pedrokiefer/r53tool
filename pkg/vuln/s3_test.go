package vuln

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestS3CNameTakeover(t *testing.T) {
	httpmock.ActivateNonDefault(cli)
	defer httpmock.DeactivateAndReset()

	ctx := context.Background()
	f := &Findings{}

	httpmock.RegisterResponder("GET", "http://test.example.com.s3-website-sa-east-1.amazonaws.com",
		httpmock.NewStringResponder(404, `<html>
		<head><title>404 Not Found</title></head>
		<body>
		<h1>404 Not Found</h1>
		<ul>
		<li>Code: NoSuchBucket</li>
		<li>Message: The specified bucket does not exist</li>
		<li>BucketName: test.example.com</li>
		<li>RequestId: WX0S0MEJN7YMTX9X</li>
		<li>HostId: C34x66Zdyh55IscVhTGe4gyak+PMOv+SAKguQNX75quFkA179U+vB9rTSRL4yiku6TRqDP7mi9g=</li>
		</ul>
		<hr/>
		</body>
		</html>`))

	records := rtypes.ResourceRecordSet{
		Name: aws.String("test.example.com"),
		Type: rtypes.RRTypeCname,
		ResourceRecords: []rtypes.ResourceRecord{
			{
				Value: aws.String("test.example.com.s3-website-sa-east-1.amazonaws.com"),
			},
		},
	}
	checkCnameS3(ctx, f, records)

	assert.Equal(t, 1, len(f.VulnerableRecords))
	assert.Equal(t, 0, len(f.MisconfigRecords))
	assert.Equal(t, map[string]int{
		"GET http://test.example.com.s3-website-sa-east-1.amazonaws.com": 1,
	}, httpmock.GetCallCountInfo())
}
