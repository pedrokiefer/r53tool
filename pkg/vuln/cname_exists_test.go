package vuln

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dig"
	"github.com/stretchr/testify/require"
)

type fakeNXResolver struct{}

func (fakeNXResolver) Resolve(ctx context.Context, domain string, t string) error {
	return &dig.ResolveError{Domain: domain, Type: "NXDOMAIN"}
}

func TestCheckCNameExists_MisconfigOnNXDOMAIN(t *testing.T) {
	t.Cleanup(func() { dig.CurrentResolver = dig.RealResolverForTest() })
	dig.CurrentResolver = fakeNXResolver{}

	f := NewFindings(ZoneMeta{Name: "zone"})
	rs := rtypes.ResourceRecordSet{
		Name:            aws.String("name.example.com"),
		Type:            rtypes.RRTypeCname,
		ResourceRecords: []rtypes.ResourceRecord{{Value: aws.String("dst.example.net")}},
	}
	checkCNameExists(context.Background(), f, rs)
	require.Len(t, f.MisconfigRecords, 1)
}

func TestCheckAliasExists_MisconfigOnNXDOMAIN(t *testing.T) {
	t.Cleanup(func() { dig.CurrentResolver = dig.RealResolverForTest() })
	dig.CurrentResolver = fakeNXResolver{}

	f := NewFindings(ZoneMeta{Name: "zone"})
	rs := rtypes.ResourceRecordSet{
		Name:        aws.String("name.example.com"),
		Type:        rtypes.RRTypeA,
		AliasTarget: &rtypes.AliasTarget{DNSName: aws.String("dst.example.net")},
	}
	checkAliasExists(context.Background(), f, rs)
	require.Len(t, f.MisconfigRecords, 1)
}
