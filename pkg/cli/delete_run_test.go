package cli

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dig"
	"github.com/pedrokiefer/route53copy/pkg/dns"
	"github.com/stretchr/testify/require"
)

func TestDelete_Run_SkipsWhenNSMatchAndNoForce(t *testing.T) {
	oldNewRM := newRouteManager
	oldDig := getNameserversFor
	t.Cleanup(func() { newRouteManager = oldNewRM; getNameserversFor = oldDig })

	nsRS := rtypes.ResourceRecordSet{
		Name:            aws.String("example.com."),
		Type:            rtypes.RRTypeNs,
		ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("ns1.example.net.")}},
	}
	fake := &fakeRouteManager{
		HostedZone:  rtypes.HostedZone{Id: aws.String("/hostedzone/Z1"), Name: aws.String("example.com.")},
		RecordsByID: map[string][]rtypes.ResourceRecordSet{"/hostedzone/Z1": {nsRS}},
	}
	newRouteManager = func(ctx context.Context, profile string, rmo *dns.RouteManagerOptions) RouteManagerAPI { return fake }
	getNameserversFor = func(domain string) ([]string, error) { return []string{"ns1.example.net"}, nil }

	a := &deleteApp{Profile: "p", Domain: "example.com."}
	err := a.Run(context.Background())
	require.NoError(t, err)
	require.False(t, fake.DeleteRecordsCalled)
	require.False(t, fake.DeleteZoneCalled)
}

func TestDelete_Run_ForceDeletesWhenNoNS(t *testing.T) {
	oldNewRM := newRouteManager
	oldDig := getNameserversFor
	oldPrompt := promptConfirm
	t.Cleanup(func() { newRouteManager = oldNewRM; getNameserversFor = oldDig; promptConfirm = oldPrompt })

	rr := rtypes.ResourceRecordSet{Name: aws.String("example.com."), Type: rtypes.RRTypeA, ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("1.2.3.4")}}}
	nsRS := rtypes.ResourceRecordSet{
		Name:            aws.String("example.com."),
		Type:            rtypes.RRTypeNs,
		ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("ns1.example.net.")}},
	}
	fake := &fakeRouteManager{
		HostedZone:  rtypes.HostedZone{Id: aws.String("/hostedzone/Z1"), Name: aws.String("example.com.")},
		RecordsByID: map[string][]rtypes.ResourceRecordSet{"/hostedzone/Z1": {rr, nsRS}},
	}
	newRouteManager = func(ctx context.Context, profile string, rmo *dns.RouteManagerOptions) RouteManagerAPI { return fake }
	getNameserversFor = func(domain string) ([]string, error) { return nil, &dig.NSRecordNotFound{Domain: domain} }
	promptConfirm = func(label string, isConfirm bool) (string, error) { return "y", nil }

	// Force skip prompt path by setting Force true
	a := &deleteApp{Profile: "p", Domain: "example.com.", Force: true}
	err := a.Run(context.Background())
	// We expect it to try deleting records and zone; our fake marks flags
	require.NoError(t, err)
	require.True(t, fake.DeleteRecordsCalled)
	require.True(t, fake.DeleteZoneCalled)
}
