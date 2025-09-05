package cli

import (
	"context"
	"time"

	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/manifoldco/promptui"
	"github.com/pedrokiefer/route53copy/pkg/dig"
	"github.com/pedrokiefer/route53copy/pkg/dns"
)

// RouteManagerAPI declares the subset of dns.RouteManager used by the CLI.
// Tests can implement this interface to stub AWS interactions.
type RouteManagerAPI interface {
	GetHostedZone(ctx context.Context, domain string) (rtypes.HostedZone, error)
	ListHostedZones(ctx context.Context) ([]rtypes.HostedZone, error)
	GetResourceRecords(ctx context.Context, zoneId string) ([]rtypes.ResourceRecordSet, error)
	GetNSRecords(ctx context.Context, zoneId string) (rtypes.ResourceRecordSet, error)
	CreateChanges(domain string, recordSets []rtypes.ResourceRecordSet) []rtypes.Change
	UpdateRecords(ctx context.Context, comment, zoneId string, changes []rtypes.Change) (*rtypes.ChangeInfo, error)
	WaitForChange(ctx context.Context, changeId string, maxWait time.Duration) error
	GetOrCreateZone(ctx context.Context, domain string) (rtypes.HostedZone, error)
	UpdateNSRecords(ctx context.Context, domain, zoneId string) (bool, error)
	DeleteRecords(ctx context.Context, zoneId string, records []rtypes.ResourceRecordSet) (string, error)
	DeleteHostedZone(ctx context.Context, zoneId string) (string, error)
	GetZoneTags(ctx context.Context, zoneID string) ([]dns.Tag, error)
	UpsertTags(ctx context.Context, zoneID string, tags []dns.Tag) error
}

// newRouteManager is a seam to allow injecting a fake RouteManager in tests.
// By default, it constructs the real dns.RouteManager, which satisfies RouteManagerAPI.
var newRouteManager = func(ctx context.Context, profile string, rmo *dns.RouteManagerOptions) RouteManagerAPI {
	return dns.NewRouteManager(ctx, profile, rmo)
}

// getNameserversFor is a seam over dig.GetNameserversFor used by some CLI commands.
var getNameserversFor = func(domain string) ([]string, error) { return dig.GetNameserversFor(domain) }

// writeBindZoneFile is a seam over dns.WriteBindZoneFile used by export.
var writeBindZoneFile = func(outputPath, zone string, records []rtypes.ResourceRecordSet) error {
	return dns.WriteBindZoneFile(outputPath, zone, records)
}

// promptConfirm wraps a confirm prompt; tests can override to auto-confirm.
var promptConfirm = func(label string, isConfirm bool) (string, error) {
	prompt := promptui.Prompt{Label: label, IsConfirm: isConfirm}
	return prompt.Run()
}
