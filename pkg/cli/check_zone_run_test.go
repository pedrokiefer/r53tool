package cli

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/stretchr/testify/require"
)

func TestCheckZone_Run_MismatchLogsButNoError(t *testing.T) {
	oldNewRM := newRouteManager
	oldDig := getNameserversFor
	t.Cleanup(func() { newRouteManager = oldNewRM; getNameserversFor = oldDig })

	fake := &fakeRouteManager{
		HostedZone: rtypes.HostedZone{Id: aws.String("/hostedzone/Z1"), Name: aws.String("example.com.")},
		NSByID: map[string]rtypes.ResourceRecordSet{
			"/hostedzone/Z1": {ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("ns2.example.net.")}}},
		},
	}
	newRouteManager = func(ctx context.Context, profile string, rmo *dns.RouteManagerOptions) RouteManagerAPI { return fake }
	getNameserversFor = func(domain string) ([]string, error) { return []string{"ns1.example.net"}, nil }

	a := &checkZoneApp{Profile: "p", Domain: "example.com."}
	err := a.Run(context.Background())
	// Mismatch should log and return nil
	require.NoError(t, err)
}
