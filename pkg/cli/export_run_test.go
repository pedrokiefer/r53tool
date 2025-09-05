package cli

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/stretchr/testify/require"
)

func TestExport_Run_DryRunSkipsWrite(t *testing.T) {
	oldNewRM := newRouteManager
	oldWB := writeBindZoneFile
	t.Cleanup(func() { newRouteManager = oldNewRM; writeBindZoneFile = oldWB })

	fake := &fakeRouteManager{
		HostedZone: rtypes.HostedZone{Id: aws.String("/hostedzone/Z1"), Name: aws.String("example.com.")},
		RecordsByID: map[string][]rtypes.ResourceRecordSet{
			"/hostedzone/Z1": {
				{Name: aws.String("example.com."), Type: rtypes.RRTypeA, ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("1.2.3.4")}}},
			},
		},
	}
	newRouteManager = func(ctx context.Context, profile string, rmo *dns.RouteManagerOptions) RouteManagerAPI { return fake }

	wrote := false
	writeBindZoneFile = func(outputPath, zone string, records []rtypes.ResourceRecordSet) error {
		wrote = true
		return nil
	}

	a := &exportApp{Profile: "p", Zone: "example.com.", Output: ""}
	dryRun = true
	err := a.Run(context.Background())
	dryRun = false
	require.NoError(t, err)
	require.False(t, wrote, "should not write when dry run")
}
