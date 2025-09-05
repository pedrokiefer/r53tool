package cli

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	rtypes "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/pedrokiefer/route53copy/pkg/dns"
)

type fakeRouteManager struct {
	HostedZone  rtypes.HostedZone
	Zones       []rtypes.HostedZone
	RecordsByID map[string][]rtypes.ResourceRecordSet
	NSByID      map[string]rtypes.ResourceRecordSet

	UpdateRecordsCalled bool
	DeleteRecordsCalled bool
	DeleteZoneCalled    bool
}

func (f *fakeRouteManager) GetHostedZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	return f.HostedZone, nil
}
func (f *fakeRouteManager) ListHostedZones(ctx context.Context) ([]rtypes.HostedZone, error) {
	return f.Zones, nil
}
func (f *fakeRouteManager) GetResourceRecords(ctx context.Context, zoneId string) ([]rtypes.ResourceRecordSet, error) {
	if f.RecordsByID == nil {
		return nil, nil
	}
	return f.RecordsByID[zoneId], nil
}
func (f *fakeRouteManager) GetNSRecords(ctx context.Context, zoneId string) (rtypes.ResourceRecordSet, error) {
	if f.NSByID == nil {
		return rtypes.ResourceRecordSet{}, nil
	}
	return f.NSByID[zoneId], nil
}
func (f *fakeRouteManager) CreateChanges(domain string, recordSets []rtypes.ResourceRecordSet) []rtypes.Change {
	return dns.NewRouteManager(context.Background(), "", &dns.RouteManagerOptions{}).CreateChanges(domain, recordSets)
}
func (f *fakeRouteManager) UpdateRecords(ctx context.Context, comment, zoneId string, changes []rtypes.Change) (*rtypes.ChangeInfo, error) {
	f.UpdateRecordsCalled = true
	return &rtypes.ChangeInfo{Id: aws.String("chg"), Status: rtypes.ChangeStatusInsync}, nil
}
func (f *fakeRouteManager) WaitForChange(ctx context.Context, changeId string, maxWait time.Duration) error {
	return nil
}
func (f *fakeRouteManager) GetOrCreateZone(ctx context.Context, domain string) (rtypes.HostedZone, error) {
	return f.HostedZone, nil
}
func (f *fakeRouteManager) UpdateNSRecords(ctx context.Context, domain, zoneId string) (bool, error) {
	return false, nil
}
func (f *fakeRouteManager) DeleteRecords(ctx context.Context, zoneId string, records []rtypes.ResourceRecordSet) (string, error) {
	f.DeleteRecordsCalled = true
	return "drchg", nil
}
func (f *fakeRouteManager) DeleteHostedZone(ctx context.Context, zoneId string) (string, error) {
	f.DeleteZoneCalled = true
	return "dzchg", nil
}
func (f *fakeRouteManager) GetZoneTags(ctx context.Context, zoneID string) ([]dns.Tag, error) {
	return nil, nil
}
func (f *fakeRouteManager) UpsertTags(ctx context.Context, zoneID string, tags []dns.Tag) error {
	return nil
}
